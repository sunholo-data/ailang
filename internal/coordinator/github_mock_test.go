package coordinator

import (
	"fmt"
	"sync"
)

// MockGitHubClient provides a mock GitHub client for testing
type MockGitHubClient struct {
	mu             sync.Mutex
	comments       map[int][]string          // issueNum -> comments
	labels         map[int][]string          // issueNum -> labels
	definedLabels  map[string]*MockLabelInfo // labelName -> info
	closedIssues   map[int]string            // issueNum -> close reason
	addCommentErr  error
	addLabelErr    error
	removeLabelErr error
	closeIssueErr  error
	getLabelsErr   error
	ensureLabelErr error
	callCounts     map[string]int
}

// MockLabelInfo represents label information for testing
type MockLabelInfo struct {
	Name        string
	Description string
	Color       string
}

// NewMockGitHubClient creates a new mock GitHub client
func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{
		comments:      make(map[int][]string),
		labels:        make(map[int][]string),
		definedLabels: make(map[string]*MockLabelInfo),
		closedIssues:  make(map[int]string),
		callCounts:    make(map[string]int),
	}
}

// AddComment adds a comment to an issue (mock)
func (m *MockGitHubClient) AddComment(repo string, issueNum int, body string) error {
	m.callCounts["AddComment"]++
	if m.addCommentErr != nil {
		return m.addCommentErr
	}
	m.comments[issueNum] = append(m.comments[issueNum], body)
	return nil
}

// AddLabelToIssue adds a label to an issue (mock)
func (m *MockGitHubClient) AddLabelToIssue(repo string, issueNum int, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCounts["AddLabelToIssue"]++
	if m.addLabelErr != nil {
		return m.addLabelErr
	}
	// Check if label is defined
	if _, ok := m.definedLabels[label]; !ok {
		return fmt.Errorf("label %q not defined", label)
	}
	// Add label if not already present
	for _, existing := range m.labels[issueNum] {
		if existing == label {
			return nil // Already has label
		}
	}
	m.labels[issueNum] = append(m.labels[issueNum], label)
	return nil
}

// RemoveLabelFromIssue removes a label from an issue (mock)
func (m *MockGitHubClient) RemoveLabelFromIssue(repo string, issueNum int, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCounts["RemoveLabelFromIssue"]++
	if m.removeLabelErr != nil {
		return m.removeLabelErr
	}
	labels := m.labels[issueNum]
	for i, l := range labels {
		if l == label {
			m.labels[issueNum] = append(labels[:i], labels[i+1:]...)
			return nil
		}
	}
	return nil
}

// CloseIssue closes an issue (mock)
func (m *MockGitHubClient) CloseIssue(repo string, issueNum int, comment string) error {
	m.callCounts["CloseIssue"]++
	if m.closeIssueErr != nil {
		return m.closeIssueErr
	}
	m.closedIssues[issueNum] = comment
	if comment != "" {
		m.comments[issueNum] = append(m.comments[issueNum], comment)
	}
	return nil
}

// GetIssueLabels returns labels on an issue (mock)
func (m *MockGitHubClient) GetIssueLabels(repo string, issueNum int) ([]string, error) {
	m.callCounts["GetIssueLabels"]++
	if m.getLabelsErr != nil {
		return nil, m.getLabelsErr
	}
	labels := m.labels[issueNum]
	if labels == nil {
		return []string{}, nil
	}
	return labels, nil
}

// EnsureLabel ensures a label exists (mock)
func (m *MockGitHubClient) EnsureLabel(repo, label, description, color string) error {
	m.callCounts["EnsureLabel"]++
	if m.ensureLabelErr != nil {
		return m.ensureLabelErr
	}
	if _, ok := m.definedLabels[label]; !ok {
		m.definedLabels[label] = &MockLabelInfo{
			Name:        label,
			Description: description,
			Color:       color,
		}
	}
	return nil
}

// GetComments returns comments posted to an issue
func (m *MockGitHubClient) GetComments(issueNum int) []string {
	return m.comments[issueNum]
}

// GetLabels returns labels on an issue (test helper)
func (m *MockGitHubClient) GetLabels(issueNum int) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	labels := m.labels[issueNum]
	// Return a copy to avoid concurrent modification
	result := make([]string, len(labels))
	copy(result, labels)
	return result
}

// GetCallCount returns the number of times a method was called
func (m *MockGitHubClient) GetCallCount(method string) int {
	return m.callCounts[method]
}

// IsClosed returns whether an issue was closed
func (m *MockGitHubClient) IsClosed(issueNum int) bool {
	_, ok := m.closedIssues[issueNum]
	return ok
}
