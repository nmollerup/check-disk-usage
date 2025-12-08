package main

import (
	"testing"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
}

func TestCheckArgs(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")
	
	// Save original config
	originalIncludeFSType := plugin.IncludeFSType
	originalExcludeFSType := plugin.ExcludeFSType
	originalIncludeFSPath := plugin.IncludeFSPath
	originalExcludeFSPath := plugin.ExcludeFSPath
	originalWarning := plugin.Warning
	originalCritical := plugin.Critical
	originalExtraTags := extraTags
	
	// Reset config after test
	defer func() {
		plugin.IncludeFSType = originalIncludeFSType
		plugin.ExcludeFSType = originalExcludeFSType
		plugin.IncludeFSPath = originalIncludeFSPath
		plugin.ExcludeFSPath = originalExcludeFSPath
		plugin.Warning = originalWarning
		plugin.Critical = originalCritical
		extraTags = originalExtraTags
	}()
	
	// Test mutually exclusive include/exclude fs type
	plugin.IncludeFSType = []string{"ext4", "xfs}"}
	plugin.ExcludeFSType = []string{"tmpfs", "devtmpfs"}
	plugin.IncludeFSPath = []string{}
	plugin.ExcludeFSPath = []string{}
	plugin.ExtraTags = []string{}
	extraTags = map[string]string{}
	i, e := checkArgs(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.Error(e)
	
	// Test mutually exclusive include/exclude fs path
	plugin.ExcludeFSType = []string{}
	plugin.IncludeFSPath = []string{"/", "/home"}
	plugin.ExcludeFSPath = []string{"/tmp"}
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.Error(e)
	
	// Test warning >= critical
	plugin.ExcludeFSPath = []string{}
	plugin.Warning = float64(80)
	plugin.Critical = float64(70)
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.Error(e)
	
	// Test valid config
	plugin.Critical = float64(90)
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateOK, i)
	assert.NoError(e)
	
	// Test valid extra tags
	plugin.ExtraTags = []string{"env=prod", "region=us-west"}
	extraTags = map[string]string{}
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateOK, i)
	assert.NoError(e)
	assert.Equal("prod", extraTags["env"])
	assert.Equal("us-west", extraTags["region"])
	
	// Test invalid extra tags
	plugin.ExtraTags = []string{"invalid-tag"}
	extraTags = map[string]string{}
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.Error(e)
}

func TestIsValidFSType(t *testing.T) {
	// Save original config
	originalIncludeFSType := plugin.IncludeFSType
	originalExcludeFSType := plugin.ExcludeFSType
	
	// Reset config after test
	defer func() {
		plugin.IncludeFSType = originalIncludeFSType
		plugin.ExcludeFSType = originalExcludeFSType
	}()
	
	t.Run("no filters - all valid", func(t *testing.T) {
		plugin.IncludeFSType = []string{}
		plugin.ExcludeFSType = []string{}
		assert.True(t, isValidFSType("ext4"))
		assert.True(t, isValidFSType("xfs"))
		assert.True(t, isValidFSType("tmpfs"))
	})
	
	t.Run("include filter - only included types valid", func(t *testing.T) {
		plugin.IncludeFSType = []string{"ext4", "xfs"}
		plugin.ExcludeFSType = []string{}
		assert.True(t, isValidFSType("ext4"))
		assert.True(t, isValidFSType("xfs"))
		assert.False(t, isValidFSType("tmpfs"))
		assert.False(t, isValidFSType("btrfs"))
	})
	
	t.Run("exclude filter - excluded types invalid", func(t *testing.T) {
		plugin.IncludeFSType = []string{}
		plugin.ExcludeFSType = []string{"tmpfs", "devtmpfs"}
		assert.True(t, isValidFSType("ext4"))
		assert.True(t, isValidFSType("xfs"))
		assert.False(t, isValidFSType("tmpfs"))
		assert.False(t, isValidFSType("devtmpfs"))
	})
}

func TestIsValidFSPath(t *testing.T) {
	// Save original config
	originalIncludeFSPath := plugin.IncludeFSPath
	originalExcludeFSPath := plugin.ExcludeFSPath
	
	// Reset config after test
	defer func() {
		plugin.IncludeFSPath = originalIncludeFSPath
		plugin.ExcludeFSPath = originalExcludeFSPath
	}()
	
	t.Run("no filters - all valid", func(t *testing.T) {
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{}
		assert.True(t, isValidFSPath("/"))
		assert.True(t, isValidFSPath("/home"))
		assert.True(t, isValidFSPath("/tmp"))
	})
	
	t.Run("include filter - only included paths valid", func(t *testing.T) {
		plugin.IncludeFSPath = []string{"/", "/home"}
		plugin.ExcludeFSPath = []string{}
		assert.True(t, isValidFSPath("/"))
		assert.True(t, isValidFSPath("/home"))
		assert.False(t, isValidFSPath("/tmp"))
		assert.False(t, isValidFSPath("/var"))
	})
	
	t.Run("exclude filter - excluded paths invalid", func(t *testing.T) {
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{"/tmp", "/var"}
		assert.True(t, isValidFSPath("/"))
		assert.True(t, isValidFSPath("/home"))
		assert.False(t, isValidFSPath("/tmp"))
		assert.False(t, isValidFSPath("/var"))
	})
	
	t.Run("glob patterns work", func(t *testing.T) {
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{"/tmp/*", "/var/log*"}
		assert.True(t, isValidFSPath("/"))
		assert.True(t, isValidFSPath("/tmp"))
		assert.False(t, isValidFSPath("/tmp/foo"))
		assert.False(t, isValidFSPath("/var/log"))
	})
}

func TestIsReadOnly(t *testing.T) {
	t.Run("read-only mount options", func(t *testing.T) {
		assert.True(t, isReadOnly("ro"))
		assert.True(t, isReadOnly("ro,noexec,nosuid"))
		assert.True(t, isReadOnly("noexec,ro,nosuid"))
		assert.True(t, isReadOnly("read-only"))
		assert.True(t, isReadOnly("noexec,read-only,nosuid"))
	})
	
	t.Run("read-write mount options", func(t *testing.T) {
		assert.False(t, isReadOnly("rw"))
		assert.False(t, isReadOnly("rw,noexec,nosuid"))
		assert.False(t, isReadOnly(""))
	})
}

func TestContains(t *testing.T) {
	t.Run("string found in slice", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.True(t, contains(slice, "a"))
		assert.True(t, contains(slice, "b"))
		assert.True(t, contains(slice, "c"))
	})
	
	t.Run("string not found in slice", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.False(t, contains(slice, "d"))
		assert.False(t, contains(slice, ""))
	})
	
	t.Run("empty slice", func(t *testing.T) {
		slice := []string{}
		assert.False(t, contains(slice, "a"))
	})
}

func TestMetricGroupAddMetric(t *testing.T) {
	mg := &MetricGroup{
		Name:    "test.metric",
		Type:    "GAUGE",
		Comment: "Test metric",
		Metrics: []Metric{},
	}
	
	tags := map[string]string{"env": "test", "region": "us"}
	value := 42.5
	timestamp := int64(1234567890)
	
	mg.AddMetric(tags, value, timestamp)
	
	assert.Equal(t, 1, len(mg.Metrics))
	assert.Equal(t, tags, mg.Metrics[0].Tags)
	assert.Equal(t, value, mg.Metrics[0].Value)
	assert.Equal(t, timestamp, mg.Metrics[0].Timestamp)
	
	// Add another metric
	mg.AddMetric(map[string]string{"env": "prod"}, 100.0, int64(9876543210))
	assert.Equal(t, 2, len(mg.Metrics))
}

func TestMetricGroupOutput(t *testing.T) {
	t.Run("output with no tags", func(t *testing.T) {
		mg := &MetricGroup{
			Name:    "disk.usage",
			Type:    "GAUGE",
			Comment: "Disk usage metric",
			Metrics: []Metric{
				{
					Tags:      map[string]string{},
					Value:     50.5,
					Timestamp: 1234567890,
				},
			},
		}
		// Just verify it doesn't panic - output goes to stdout
		mg.Output()
	})
	
	t.Run("output with single tag", func(t *testing.T) {
		mg := &MetricGroup{
			Name:    "disk.free",
			Type:    "GAUGE",
			Comment: "Free disk space",
			Metrics: []Metric{
				{
					Tags:      map[string]string{"mountpoint": "/"},
					Value:     100.0,
					Timestamp: 1234567890,
				},
			},
		}
		mg.Output()
	})
	
	t.Run("output with multiple tags", func(t *testing.T) {
		mg := &MetricGroup{
			Name:    "disk.percent_used",
			Type:    "GAUGE",
			Comment: "Percentage of disk used",
			Metrics: []Metric{
				{
					Tags: map[string]string{
						"mountpoint": "/home",
						"env":        "test",
						"region":     "us-west",
					},
					Value:     75.5,
					Timestamp: 1234567890,
				},
			},
		}
		mg.Output()
	})
	
	t.Run("output with multiple metrics", func(t *testing.T) {
		mg := &MetricGroup{
			Name:    "disk.critical",
			Type:    "GAUGE",
			Comment: "Critical disk status",
			Metrics: []Metric{
				{
					Tags:      map[string]string{"mountpoint": "/"},
					Value:     0,
					Timestamp: 1234567890,
				},
				{
					Tags:      map[string]string{"mountpoint": "/home"},
					Value:     1,
					Timestamp: 1234567890,
				},
			},
		}
		mg.Output()
	})
	
	t.Run("output with dots in name", func(t *testing.T) {
		mg := &MetricGroup{
			Name:    "disk.free.inodes",
			Type:    "GAUGE",
			Comment: "Free inodes",
			Metrics: []Metric{
				{
					Tags:      map[string]string{},
					Value:     1000,
					Timestamp: 1234567890,
				},
			},
		}
		// Dots should be replaced with underscores in the metric name
		mg.Output()
	})
}

func TestExecuteCheck(t *testing.T) {
	// Save original config
	originalWarning := plugin.Warning
	originalCritical := plugin.Critical
	originalInodesWarning := plugin.InodesWarning
	originalInodesCritical := plugin.InodesCritical
	originalIncludeFSType := plugin.IncludeFSType
	originalExcludeFSType := plugin.ExcludeFSType
	originalIncludeFSPath := plugin.IncludeFSPath
	originalExcludeFSPath := plugin.ExcludeFSPath
	originalIncludePseudo := plugin.IncludePseudo
	originalIncludeReadOnly := plugin.IncludeReadOnly
	originalFailOnError := plugin.FailOnError
	originalMetricsMode := plugin.MetricsMode
	originalHumanReadable := plugin.HumanReadable
	originalExtraTags := extraTags
	
	// Reset config after test
	defer func() {
		plugin.Warning = originalWarning
		plugin.Critical = originalCritical
		plugin.InodesWarning = originalInodesWarning
		plugin.InodesCritical = originalInodesCritical
		plugin.IncludeFSType = originalIncludeFSType
		plugin.ExcludeFSType = originalExcludeFSType
		plugin.IncludeFSPath = originalIncludeFSPath
		plugin.ExcludeFSPath = originalExcludeFSPath
		plugin.IncludePseudo = originalIncludePseudo
		plugin.IncludeReadOnly = originalIncludeReadOnly
		plugin.FailOnError = originalFailOnError
		plugin.MetricsMode = originalMetricsMode
		plugin.HumanReadable = originalHumanReadable
		extraTags = originalExtraTags
	}()
	
	event := corev2.FixtureEvent("entity1", "check1")
	
	t.Run("basic check execution - normal thresholds", func(t *testing.T) {
		plugin.Warning = 85.0
		plugin.Critical = 95.0
		plugin.InodesWarning = 85.0
		plugin.InodesCritical = 95.0
		plugin.IncludeFSType = []string{}
		plugin.ExcludeFSType = []string{}
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{}
		plugin.IncludePseudo = false
		plugin.IncludeReadOnly = false
		plugin.FailOnError = false
		plugin.MetricsMode = false
		plugin.HumanReadable = false
		extraTags = map[string]string{}
		
		status, err := executeCheck(event)
		assert.NoError(t, err)
		// Status will be OK, WARNING, or CRITICAL depending on the actual disk state
		assert.True(t, status == sensu.CheckStateOK || status == sensu.CheckStateWarning || status == sensu.CheckStateCritical)
	})
	
	t.Run("metrics mode execution", func(t *testing.T) {
		plugin.Warning = 85.0
		plugin.Critical = 95.0
		plugin.InodesWarning = 85.0
		plugin.InodesCritical = 95.0
		plugin.IncludeFSType = []string{}
		plugin.ExcludeFSType = []string{}
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{}
		plugin.IncludePseudo = false
		plugin.IncludeReadOnly = false
		plugin.FailOnError = false
		plugin.MetricsMode = true
		plugin.HumanReadable = false
		extraTags = map[string]string{}
		
		status, err := executeCheck(event)
		assert.NoError(t, err)
		assert.True(t, status == sensu.CheckStateOK || status == sensu.CheckStateWarning || status == sensu.CheckStateCritical)
	})
	
	t.Run("human readable mode", func(t *testing.T) {
		plugin.Warning = 85.0
		plugin.Critical = 95.0
		plugin.InodesWarning = 85.0
		plugin.InodesCritical = 95.0
		plugin.IncludeFSType = []string{}
		plugin.ExcludeFSType = []string{}
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{}
		plugin.IncludePseudo = false
		plugin.IncludeReadOnly = false
		plugin.FailOnError = false
		plugin.MetricsMode = false
		plugin.HumanReadable = true
		extraTags = map[string]string{}
		
		status, err := executeCheck(event)
		assert.NoError(t, err)
		assert.True(t, status == sensu.CheckStateOK || status == sensu.CheckStateWarning || status == sensu.CheckStateCritical)
	})
	
	t.Run("with extra tags", func(t *testing.T) {
		plugin.Warning = 85.0
		plugin.Critical = 95.0
		plugin.InodesWarning = 85.0
		plugin.InodesCritical = 95.0
		plugin.IncludeFSType = []string{}
		plugin.ExcludeFSType = []string{}
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{}
		plugin.IncludePseudo = false
		plugin.IncludeReadOnly = false
		plugin.FailOnError = false
		plugin.MetricsMode = false
		plugin.HumanReadable = false
		extraTags = map[string]string{"env": "test", "region": "us-west"}
		
		status, err := executeCheck(event)
		assert.NoError(t, err)
		assert.True(t, status == sensu.CheckStateOK || status == sensu.CheckStateWarning || status == sensu.CheckStateCritical)
	})
	
	t.Run("with very low thresholds to trigger warnings", func(t *testing.T) {
		plugin.Warning = 1.0
		plugin.Critical = 99.0
		plugin.InodesWarning = 1.0
		plugin.InodesCritical = 99.0
		plugin.IncludeFSType = []string{}
		plugin.ExcludeFSType = []string{}
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{}
		plugin.IncludePseudo = false
		plugin.IncludeReadOnly = false
		plugin.FailOnError = false
		plugin.MetricsMode = false
		plugin.HumanReadable = false
		extraTags = map[string]string{}
		
		status, err := executeCheck(event)
		assert.NoError(t, err)
		// With very low thresholds, we expect at least a warning
		assert.True(t, status == sensu.CheckStateWarning || status == sensu.CheckStateCritical)
	})
	
	t.Run("with include pseudo filesystems", func(t *testing.T) {
		plugin.Warning = 85.0
		plugin.Critical = 95.0
		plugin.InodesWarning = 85.0
		plugin.InodesCritical = 95.0
		plugin.IncludeFSType = []string{}
		plugin.ExcludeFSType = []string{}
		plugin.IncludeFSPath = []string{}
		plugin.ExcludeFSPath = []string{}
		plugin.IncludePseudo = true
		plugin.IncludeReadOnly = false
		plugin.FailOnError = false
		plugin.MetricsMode = false
		plugin.HumanReadable = false
		extraTags = map[string]string{}
		
		status, err := executeCheck(event)
		assert.NoError(t, err)
		assert.True(t, status == sensu.CheckStateOK || status == sensu.CheckStateWarning || status == sensu.CheckStateCritical)
	})
}
