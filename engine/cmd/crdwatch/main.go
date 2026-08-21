//go:build windows

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/khiemnguyen/remotune/engine/internal/crd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: crdwatch <logfile>")
		os.Exit(2)
	}
	logPath := os.Args[1]

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open log:", err)
		os.Exit(1)
	}
	defer f.Close()

	logf := func(format string, args ...any) {
		line := fmt.Sprintf("[%s] "+format+"\n", append([]any{time.Now().Format(time.RFC3339Nano)}, args...)...)
		f.WriteString(line)
		f.Sync()
	}

	logf("crdwatch starting")

	boot, err := crd.Bootstrap()
	if err != nil {
		logf("FATAL bootstrap: %v", err)
		os.Exit(1)
	}
	logf("bootstrap state=%s activeSessions=%d currentHostPID=%d droppedDisconnects=%d bookmarkLen=%d",
		boot.Snapshot.State, len(boot.Snapshot.ActiveSessions), boot.Snapshot.CurrentHostPID,
		boot.Snapshot.DroppedDisconnects, len(boot.Bookmark))
	for _, s := range boot.Snapshot.ActiveSessions {
		logf("bootstrap active session=%s pid=%d", s.SessionID, s.ProcessID)
	}

	sub, err := crd.SubscribeAfterBookmark(boot.Bookmark)
	if err != nil {
		logf("FATAL subscribe: %v", err)
		os.Exit(1)
	}
	defer sub.Close()

	logf("subscribed; polling. waiting for connect/disconnect transitions...")

	for {
		transitions, skipped, err := sub.Poll(32, 2000)
		if err != nil {
			logf("poll error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		if skipped > 0 {
			logf("poll: %d event(s) skipped (malformed)", skipped)
		}
		for _, t := range transitions {
			logf("TRANSITION kind=%s rec=%d pid=%d session=%s time=%s",
				t.Kind, t.RecordID, t.ProcessID, t.SessionID, t.Time.Format(time.RFC3339))
		}
	}
}
