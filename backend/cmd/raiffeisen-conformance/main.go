package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/integrations/raiffeisen"
)

// Conformance retains private responses locally. It has no database or admission capability.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Conformance incomplete; inspect private state before retrying:", err)
		os.Exit(1)
	}
}
func privateFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("private file required")
	}
	return os.ReadFile(path)
}
func save(directory, name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".pending-*")
	if err != nil {
		return err
	}
	nameTemp := file.Name()
	defer os.Remove(nameTemp)
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(nameTemp, filepath.Join(directory, name)); err != nil {
		return err
	}
	dir, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
func run() error {
	directory := flag.String("directory", "", "existing private research directory")
	secretFile := flag.String("client-secret-file", "", "private operator client secret file")
	refresh := flag.Bool("refresh", false, "rotate the latest persisted refresh token once")
	reportState := flag.String("report-state", "", "existing private report-state file to read")
	from := flag.String("from", "", "request a new historical report from this date")
	to := flag.String("to", "", "last date of the new historical report")
	flag.Parse()
	if *directory == "" {
		return errors.New("directory required")
	}
	root, err := filepath.EvalSymlinks(*directory)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.New("private directory required")
	}
	for p := root; ; p = filepath.Dir(p) {
		if _, err = os.Lstat(filepath.Join(p, ".git")); err == nil {
			return errors.New("evidence cannot enter git")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	lock, err := os.OpenFile(filepath.Join(root, ".native-conformance.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	markerBytes, err := privateFile(filepath.Join(root, "refresh-state.json"))
	if err != nil {
		return err
	}
	var marker struct {
		Phase string `json:"phase"`
	}
	if json.Unmarshal(markerBytes, &marker) != nil || marker.Phase != "complete" {
		return errors.New("rotation unresolved")
	}
	tokenBytes, err := privateFile(filepath.Join(root, "tokens.json"))
	if err != nil {
		return err
	}
	defer clear(tokenBytes)
	var tokens raiffeisen.Tokens
	if json.Unmarshal(tokenBytes, &tokens) != nil || tokens.Validate() != nil {
		return errors.New("tokens invalid")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := raiffeisen.NewClient(nil)
	if *refresh {
		secret, err := privateFile(*secretFile)
		if err != nil {
			return err
		}
		defer clear(secret)
		// Audience is only an existing registration hint. The bank authenticates it with the client secret.
		// This bootstrap path never establishes an application principal or verifies an OIDC login.
		var tokenParts struct {
			Audience json.RawMessage `json:"aud"`
		}
		segments := strings.Split(tokens.ID, ".")
		if len(segments) != 3 {
			return errors.New("registration hint absent")
		}
		claims, err := base64.RawURLEncoding.DecodeString(segments[1])
		if err != nil {
			return err
		}
		if json.Unmarshal(claims, &tokenParts) != nil {
			return errors.New("registration hint invalid")
		}
		clear(claims)
		var clientID string
		if json.Unmarshal(tokenParts.Audience, &clientID) != nil {
			var ids []string
			if json.Unmarshal(tokenParts.Audience, &ids) != nil || len(ids) != 1 {
				return errors.New("registration ambiguous")
			}
			clientID = ids[0]
		}
		attempt := uuid.NewString()
		if err = save(root, "refresh-state.json", map[string]string{"phase": "pending", "attempt": attempt}); err != nil {
			return err
		}
		fresh, err := client.Refresh(ctx, clientID, strings.TrimSpace(string(secret)), tokens.Refresh)
		if err != nil {
			return err
		}
		if err = save(root, "tokens.json", fresh); err != nil {
			return err
		}
		if err = save(root, "refresh-state.json", map[string]string{"phase": "complete", "attempt": attempt}); err != nil {
			return err
		}
		tokens = fresh
	}
	accounts, raw, err := client.Accounts(ctx, tokens)
	if err != nil {
		return fmt.Errorf("accounts read: %w", err)
	}
	if err = save(root, "native-accounts.json", map[string]any{"status": 200, "body_base64": base64.StdEncoding.EncodeToString(raw), "read_at": time.Now().UTC().Format(time.RFC3339Nano)}); err != nil {
		return err
	}
	count, facts, gaps := len(accounts), 0, 0
	if *from != "" || *to != "" {
		if len(accounts) != 1 {
			return errors.New("account binding ambiguous")
		}
		zone, _ := time.LoadLocation("Europe/Moscow")
		start, err := time.ParseInLocation(time.DateOnly, *from, zone)
		if err != nil {
			return err
		}
		end, err := time.ParseInLocation(time.DateOnly, *to, zone)
		if err != nil {
			return err
		}
		if start.After(end) || end.Format(time.DateOnly) >= time.Now().In(zone).Format(time.DateOnly) || end.Sub(start) > 90*24*time.Hour {
			return errors.New("historical range required")
		}
		name := "native-statement-" + *from + "-" + *to + ".json"
		path := filepath.Join(root, name)
		if _, err = os.Lstat(path); os.IsNotExist(err) {
			if err = save(root, name, map[string]string{"phase": "pending"}); err != nil {
				return err
			}
			id, response, err := client.CreateReport(ctx, tokens, accounts[0], start, end, time.Now())
			if saveErr := save(root, "native-report-create.json", map[string]any{"body_base64": base64.StdEncoding.EncodeToString(response), "read_at": time.Now().UTC().Format(time.RFC3339Nano)}); saveErr != nil {
				return saveErr
			}
			if err != nil {
				return err
			}
			if err = save(root, name, map[string]string{"phase": "accepted", "report_id": id}); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		*reportState = path
	}
	if *reportState != "" {
		data, err := privateFile(*reportState)
		if err != nil {
			return err
		}
		var report struct {
			Phase string `json:"phase"`
			ID    string `json:"report_id"`
		}
		if json.Unmarshal(data, &report) != nil || report.Phase != "accepted" || len(accounts) != 1 {
			return errors.New("report binding unavailable")
		}
		var file, status []byte
		for attempt := 0; attempt < 10; attempt++ {
			file, status, err = client.Report(ctx, tokens, report.ID)
			if !errors.Is(err, raiffeisen.ErrReportPending) {
				break
			}
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		if saveErr := save(root, "native-report-status.json", map[string]any{"body_base64": base64.StdEncoding.EncodeToString(status), "read_at": time.Now().UTC().Format(time.RFC3339Nano)}); saveErr != nil {
			return saveErr
		}
		if err != nil {
			return err
		}
		if err = save(root, "native-report.json", map[string]any{"status": 200, "body_base64": base64.StdEncoding.EncodeToString(file), "status_body_base64": base64.StdEncoding.EncodeToString(status), "read_at": time.Now().UTC().Format(time.RFC3339Nano)}); err != nil {
			return err
		}
		normalized, err := raiffeisen.Normalize(file, accounts[0], report.ID)
		if err != nil {
			return err
		}
		facts, gaps = len(normalized.Facts), len(normalized.Gaps)
	}
	fmt.Printf("accounts_http=200 accounts=%d statement_facts=%d coverage_gaps=%d private_evidence_saved=true admission_changed=false\n", count, facts, gaps)
	return nil
}
