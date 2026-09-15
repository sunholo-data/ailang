package config

// Pub/Sub topic naming.
const EnvTopicPrefix = "AILANG_TOPIC_PREFIX"

// DefaultTopicPrefix is the prefix of every AILANG Pub/Sub topic when
// AILANG_TOPIC_PREFIX is unset; terraform sets ailang / ailang-dev per
// environment.
const DefaultTopicPrefix = "ailang"

var pubsubVars = []Var{
	{EnvTopicPrefix, DefaultTopicPrefix, AreaPubSub, "Prefix of every Pub/Sub topic name (<prefix>-messages, <prefix>-tasks, ...); terraform sets it per environment."},
}

// TopicPrefix returns AILANG_TOPIC_PREFIX, default DefaultTopicPrefix.
func TopicPrefix() string { return getOr(EnvTopicPrefix) }
