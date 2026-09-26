package firestore

import (
	"context"
	"net"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	gcfirestore "cloud.google.com/go/firestore"
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	obs "github.com/sunholo-data/ailang/internal/observatory"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Exercise the real Firestore SDK transport against an in-memory RPC fixture.
// No ADC, emulator installation, external endpoint or production records.
type readFixture struct {
	pb.UnimplementedFirestoreServer
	queries    chan *pb.StructuredQuery
	failAgents bool
}

func (f *readFixture) RunQuery(req *pb.RunQueryRequest, stream pb.Firestore_RunQueryServer) error {
	q := req.GetStructuredQuery()
	f.queries <- q
	collection := q.GetFrom()[0].GetCollectionId()
	ids := []string{"c5", "c4", "c3", "c2", "c1"}
	if collection == collObsSpans {
		ids = []string{"a", "b", "c"}
	}
	if collection == collObsChainStages {
		if f.failAgents {
			return status.Error(codes.PermissionDenied, "fixture denied")
		}
		ids = []string{"c4", "c2", "c1"}
	}
	start := min(int(q.GetOffset()), len(ids))
	end := len(ids)
	if q.Limit != nil {
		end = min(start+int(q.Limit.Value), end)
	}
	for _, id := range ids[start:end] {
		fields := map[string]*pb.Value{
			"id":         {ValueType: &pb.Value_StringValue{StringValue: id}},
			"created_at": {ValueType: &pb.Value_TimestampValue{TimestampValue: timestamppb.New(time.Unix(100, 0))}},
			"start_time": {ValueType: &pb.Value_TimestampValue{TimestampValue: timestamppb.New(time.Unix(100, 0))}},
		}
		if collection == collObsChainStages {
			fields["chain_id"] = &pb.Value{ValueType: &pb.Value_StringValue{StringValue: id}}
		}
		if err := stream.Send(&pb.RunQueryResponse{Document: &pb.Document{Name: req.Parent + "/" + collection + "/" + id, Fields: fields, CreateTime: timestamppb.New(time.Unix(100, 0)), UpdateTime: timestamppb.New(time.Unix(100, 0))}}); err != nil {
			return err
		}
	}
	return nil
}
func fixtureStore(t *testing.T, failAgents bool) (*ObservatoryStore, *readFixture) {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	fixture := &readFixture{queries: make(chan *pb.StructuredQuery, 32), failAgents: failAgents}
	pb.RegisterFirestoreServer(server, fixture)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///fixture", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client, err := gcfirestore.NewClient(context.Background(), "fixture", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return NewObservatoryStore(&Client{fs: client, projectID: "fixture"}), fixture
}
func TestChainReadPagingContract(t *testing.T) {
	store, f := fixtureStore(t, false)
	after := time.Unix(0, 0)
	got, err := store.ListChains(context.Background(), obs.ChainListOptions{Limit: 2, Offset: 2, CreatedAfter: &after, Status: obs.ChainStatusCompleted, SourceType: "message", WorkspaceID: "workspace-a", GitHubRepo: "owner/repo"})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	if !reflect.DeepEqual(ids, []string{"c3", "c2"}) {
		t.Fatalf("page IDs=%v", ids)
	}
	q := <-f.queries
	assertOrder(t, q, []string{"created_at", "__name__"}, pb.StructuredQuery_DESCENDING)
	assertWireFilters(t, q, []*pb.StructuredQuery_FieldFilter{
		wireStringFilter("status", "completed"), wireStringFilter("source_type", "message"),
		wireStringFilter("workspace_id", "workspace-a"), wireStringFilter("github_repo", "owner/repo"),
		wireTimeFilter("created_at", pb.StructuredQuery_FieldFilter_GREATER_THAN, after),
	})
}
func TestChainAgentFilterBeforePaging(t *testing.T) {
	store, _ := fixtureStore(t, false)
	got, err := store.ListChains(context.Background(), obs.ChainListOptions{AgentID: "worker", Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "c2" {
		t.Fatalf("agent page=%+v", got)
	}
}
func TestChainAgentReadFailureIsNotEmptySuccess(t *testing.T) {
	store, _ := fixtureStore(t, true)
	got, err := store.ListChains(context.Background(), obs.ChainListOptions{AgentID: "worker", Limit: 1})
	if status.Code(err) != codes.PermissionDenied || got != nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
func TestSpanReadPagingContract(t *testing.T) {
	store, f := fixtureStore(t, false)
	after, before := time.Unix(50, 123), time.Unix(150, 456)
	got, err := store.ListSpans(context.Background(), obs.SpanListOptions{Limit: 1, Offset: 1, TraceID: "trace-a", Status: "error", StartAfter: after, StartBefore: before})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("span page=%v", got)
	}
	q := <-f.queries
	assertOrder(t, q, []string{"start_time", "__name__"}, pb.StructuredQuery_ASCENDING)
	assertWireFilters(t, q, []*pb.StructuredQuery_FieldFilter{
		wireStringFilter("trace_id", "trace-a"), wireStringFilter("status", "error"),
		wireTimeFilter("start_time", pb.StructuredQuery_FieldFilter_GREATER_THAN_OR_EQUAL, after),
		wireTimeFilter("start_time", pb.StructuredQuery_FieldFilter_LESS_THAN_OR_EQUAL, before),
	})
}
func TestReadOptionsRejectSilentlyIgnoredFilters(t *testing.T) {
	store := &ObservatoryStore{}
	for _, opts := range []obs.SpanListOptions{{Workspace: "/fixture"}, {WorkspaceID: "ws"}, {Offset: -1}} {
		if _, err := store.ListSpans(context.Background(), opts); err == nil {
			t.Fatalf("accepted unsupported/invalid options %+v", opts)
		}
	}
	if _, err := store.ListChains(context.Background(), obs.ChainListOptions{Offset: -1}); err == nil {
		t.Fatal("negative offset accepted")
	}
}
func assertOrder(t *testing.T, q *pb.StructuredQuery, fields []string, direction pb.StructuredQuery_Direction) {
	t.Helper()
	var got []string
	for _, o := range q.OrderBy {
		got = append(got, o.Field.FieldPath)
		if o.Direction != direction {
			t.Fatalf("order direction=%v", o)
		}
	}
	if !reflect.DeepEqual(got, fields) {
		t.Fatalf("order=%v want=%v", got, fields)
	}
}

func TestSQLiteAndFirestoreFixedCohortPagesAgree(t *testing.T) {
	ctx := context.Background()
	sqlite, err := obs.NewSQLiteBackendFromPath(filepath.Join(t.TempDir(), "observatory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.Close()
	for _, id := range []string{"c1", "c2", "c3", "c4", "c5"} {
		if _, err := sqlite.CreateChain(ctx, &obs.ChainCreateRequest{ID: id, SourceType: obs.ChainSourceManual}); err != nil {
			t.Fatal(err)
		}
		if _, err := sqlite.DB().Exec("UPDATE execution_chains SET created_at=? WHERE id=?", time.Unix(100, 0).UTC(), id); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"c1", "c2", "c4"} {
		if _, err := sqlite.CreateStage(ctx, &obs.StageCreateRequest{ID: "stage-" + id, ChainID: id, AgentID: "worker"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, opts := range []obs.ChainListOptions{{Limit: 2}, {Limit: 2, Offset: 2}, {Limit: 2, Offset: 4}, {Limit: 2, Offset: 9}, {AgentID: "worker", Limit: 1, Offset: 1}} {
		cloud, _ := fixtureStore(t, false)
		localRows, err := sqlite.ListChains(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		cloudRows, err := cloud.ListChains(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		if localRows == nil || cloudRows == nil {
			t.Fatalf("successful pages must not be nil: opts=%+v local=%v cloud=%v", opts, localRows, cloudRows)
		}
		ids := func(rows []*obs.ChainSummary) []string {
			result := []string{}
			for _, r := range rows {
				result = append(result, r.ID)
			}
			return result
		}
		if !reflect.DeepEqual(ids(localRows), ids(cloudRows)) {
			t.Fatalf("options=%+v SQLite=%v Firestore=%v", opts, ids(localRows), ids(cloudRows))
		}
	}
	for _, id := range []string{"c", "a", "b"} {
		if err := sqlite.CreateSpan(ctx, &obs.Span{ID: id, TraceID: "trace", Name: "fixture", StageID: "stage-c1", ChainID: "c1", StartTime: time.Unix(100, 0).UTC()}); err != nil {
			t.Fatal(err)
		}
	}
	for _, opts := range []obs.SpanListOptions{{Limit: 1, Offset: 1}, {Offset: 1}, {Limit: 1, Offset: 9}} {
		cloud, _ := fixtureStore(t, false)
		localRows, err := sqlite.ListSpans(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		cloudRows, err := cloud.ListSpans(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		if localRows == nil || cloudRows == nil {
			t.Fatalf("successful pages must not be nil: opts=%+v local=%v cloud=%v", opts, localRows, cloudRows)
		}
		ids := func(rows []*obs.Span) []string {
			result := []string{}
			for _, r := range rows {
				result = append(result, r.ID)
			}
			return result
		}
		if !reflect.DeepEqual(ids(localRows), ids(cloudRows)) {
			t.Fatalf("options=%+v SQLite=%v Firestore=%v", opts, ids(localRows), ids(cloudRows))
		}
	}
	for _, offset := range []int{0, 1, 2, 9} {
		cloud, fixture := fixtureStore(t, false)
		localPage, err := sqlite.GetSpanLitesByStageID(ctx, "stage-c1", 1, offset)
		if err != nil {
			t.Fatal(err)
		}
		cloudPage, err := cloud.GetSpanLitesByStageID(ctx, "stage-c1", 1, offset)
		if err != nil {
			t.Fatal(err)
		}
		if localPage.Spans == nil || cloudPage.Spans == nil {
			t.Fatalf("successful stage pages must not be nil: offset=%d", offset)
		}
		ids := func(rows []*obs.SpanLite) []string {
			result := []string{}
			for _, row := range rows {
				result = append(result, row.ID)
			}
			return result
		}
		if !reflect.DeepEqual(ids(localPage.Spans), ids(cloudPage.Spans)) || localPage.Total != 3 || cloudPage.Total != 3 || localPage.Offset != offset || cloudPage.Offset != offset || localPage.Limit != 1 || cloudPage.Limit != 1 {
			t.Fatalf("stage offset=%d local=%+v ids=%v cloud=%+v ids=%v", offset, localPage, ids(localPage.Spans), cloudPage, ids(cloudPage.Spans))
		}
		countQuery, pageQuery := <-fixture.queries, <-fixture.queries
		assertWireFilters(t, countQuery, []*pb.StructuredQuery_FieldFilter{wireStringFilter("stage_id", "stage-c1")})
		assertWireFilters(t, pageQuery, []*pb.StructuredQuery_FieldFilter{wireStringFilter("stage_id", "stage-c1")})
		assertOrder(t, pageQuery, []string{"start_time", "__name__"}, pb.StructuredQuery_ASCENDING)
	}

}

func wireStringFilter(field, value string) *pb.StructuredQuery_FieldFilter {
	return &pb.StructuredQuery_FieldFilter{Field: &pb.StructuredQuery_FieldReference{FieldPath: field}, Op: pb.StructuredQuery_FieldFilter_EQUAL, Value: &pb.Value{ValueType: &pb.Value_StringValue{StringValue: value}}}
}
func wireTimeFilter(field string, op pb.StructuredQuery_FieldFilter_Operator, value time.Time) *pb.StructuredQuery_FieldFilter {
	return &pb.StructuredQuery_FieldFilter{Field: &pb.StructuredQuery_FieldReference{FieldPath: field}, Op: op, Value: &pb.Value{ValueType: &pb.Value_TimestampValue{TimestampValue: timestamppb.New(value)}}}
}
func assertWireFilters(t *testing.T, q *pb.StructuredQuery, want []*pb.StructuredQuery_FieldFilter) {
	t.Helper()
	var got []*pb.StructuredQuery_FieldFilter
	var visit func(*pb.StructuredQuery_Filter)
	visit = func(f *pb.StructuredQuery_Filter) {
		if field := f.GetFieldFilter(); field != nil {
			got = append(got, field)
			return
		}
		composite := f.GetCompositeFilter()
		if composite.GetOp() != pb.StructuredQuery_CompositeFilter_AND {
			t.Fatalf("expected AND filters: %v", f)
		}
		for _, child := range composite.GetFilters() {
			visit(child)
		}
	}
	visit(q.GetWhere())
	if len(got) != len(want) {
		t.Fatalf("filters=%v want=%v", got, want)
	}
	for _, expected := range want {
		found := false
		for _, actual := range got {
			if proto.Equal(actual, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing filter %v; got=%v", expected, got)
		}
	}
}
