// ABOUTME: Tests for store domain models and their small helpers.
// ABOUTME: Keeps enum validity logic honest.
package store

import "testing"

func TestJobStatusValid(t *testing.T) {
	valid := []JobStatus{JobQueued, JobClaimed, JobRunning, JobDone, JobFailed, JobExpired}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("%q should be valid", s)
		}
	}
	if JobStatus("nope").Valid() {
		t.Error(`"nope" should be invalid`)
	}
}

func TestJobTypeValid(t *testing.T) {
	valid := []JobType{JobShell, JobReadFile, JobWriteFile, JobListDir, JobScreenshot, JobBackground}
	for _, jt := range valid {
		if !jt.Valid() {
			t.Errorf("%q should be valid", jt)
		}
	}
	if JobType("frobnicate").Valid() {
		t.Error("frobnicate should be invalid")
	}
}
