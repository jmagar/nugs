# AI Review Content from PR #1

**Extracted from PR:** https://github.com/jmagar/nugs/pull/1
**Total items found:** 14
**Items filtered out:** 19

---

- [x] [AI PROMPT - cron/cmd/api/main.go:51]
In cron/cmd/api/main.go around line 51, the startup log uses an emoji and should
be gated so it only appears in non-production environments; update the code to
check an environment variable (e.g., APP_ENV or GO_ENV) or a dedicated flag like
HOT_RELOAD_LOGS (falling back to "production" if unset) and only emit the
emoji/log when the env is not "production" (or when HOT_RELOAD_LOGS is
explicitly true); ensure production defaults remain unchanged and consider using
the existing logger/info-level call without emoji for production.

---

- [x] [AI PROMPT - cron/internal/api/handlers/webhook.go:101]
In cron/internal/api/handlers/webhook.go around lines 94-101 (and similarly
118-129) the SELECT is including w.total_deliveries and w.failed_deliveries even
though aggregated totals (COUNT(...) as total_fired, success_count) are computed
from webhook_deliveries; remove w.total_deliveries and w.failed_deliveries from
the SELECT and from the GROUP BY to avoid unnecessary grouping noise, and remove
any subsequent Scan/field usages that read those two columns so only the
aggregated fields are scanned into the result struct.

---

- [x] [AI PROMPT - cron/internal/api/handlers/webhook.go:193]
In cron/internal/api/handlers/webhook.go around lines 184-193 (and similarly
200-206), the GetWebhook SQL selects total_deliveries and failed_deliveries but
those columns are unused and force them into the GROUP BY and Scan; remove
total_deliveries and failed_deliveries from the SELECT clause and GROUP BY, and
update the destination Scan to stop expecting those fields so the query matches
the list endpoint and avoids unused columns.

---

- [x] [AI PROMPT - cron/internal/services/scheduler.go:647]
In cron/internal/services/scheduler.go around line 606, add a helper method to
check for the existence of a column on a table (SQLite) so callers can rely on
schema checks instead of brittle error-string matching; implement a method on
SchedulerService named hasScheduleColumn(table, column string) that runs a
PRAGMA table_info query (SELECT COUNT(*) FROM pragma_table_info('<table>') WHERE
name = '<column>'), scans the result into an int, returns false on any
query/scan error, and returns true only if the count is greater than zero; place
this function outside the reviewed ranges in the same file and ensure it uses
s.DB for the query and appropriate return semantics.

---

- [x] [COMMITTABLE SUGGESTION - coderabbitai[bot] - cron/internal/services/scheduler.go:682]
// Derive type from parameters (if present) when column is absent
		if !hasTypeColumn {
			if parameters.Valid && parameters.String != "" {
				var p map[string]interface{}
				if err := json.Unmarshal([]byte(parameters.String), &p); err == nil {
					if t, ok := p["type"].(string); ok && t != "" {
						schedule.Type = t
					}
				}
			}
			if schedule.Type == "" {
				schedule.Type = "manual"
			}
		}

---

- [x] [AI PROMPT - cron/internal/services/webhook.go:57]
In cron/internal/services/webhook.go around lines 54-56, the json.Marshal call
swallows the error which can result in corrupted/empty events being written to
the DB; change to capture the error (eventsJSON, err :=
json.Marshal(req.Events)) and handle it instead of ignoring it — if marshal
fails return or propagate a descriptive error (e.g. "failed to marshal events:
%w") or log and abort storing these events, so the DB never gets invalid
payloads; ensure necessary imports (fmt or errors) are present for error
wrapping/return.

---

- [x] [AI PROMPT - cron/internal/services/webhook.go:88]
In cron/internal/services/webhook.go around lines 84 to 87, the update path only
reads req.Name but API now expects Description (with Name as optional fallback);
change the logic to prefer req.Description when present and only use req.Name if
Description is nil, and append "description = ?" and the chosen value to
updates/args accordingly so the DB update uses the new Description field while
preserving backward compatibility with Name.

---

- [x] [AI PROMPT - cron/Makefile:108]
In cron/Makefile around lines 104 to 108, the clean target currently removes
individual binaries by name which won't catch new CLIs; change the clean rule to
remove the entire bin directory (e.g., use rm -rf bin/) instead of enumerating
files, while keeping the removal of coverage files and tmp/ intact so clean
removes bin/, coverage.out, coverage.html, and tmp/.

---

- [x] [AI PROMPT - .gitignore:79]
In .gitignore around lines 78 to 79, the pattern "*-fixes.md" is too broad and
may unintentionally ignore files outside the intended docs/tracker area; replace
the global pattern with a scoped path such as "docs/*-fixes.md" (or the specific
folder you use for tracker/notes, e.g., "tracker/*-fixes.md") so only the
intended files are ignored and other similarly named files elsewhere in the repo
are not masked.

---

- [x] [AI PROMPT - cron/cmd/api/main.go:148]
In cron/cmd/api/main.go around lines 145 to 148, after iterating over rows and
handling per-row Scan errors you must check rows.Err() once after the loop to
catch any deferred iterator/driver errors; add a post-loop check like "if err :=
rows.Err(); err != nil { log.Printf(\"Error iterating rows: %v\", err) }" (or
use your project's logger) so any iteration errors are logged/reported.

---

- [x] [AI PROMPT - cron/cmd/gap_report/main.go:170]
In cron/cmd/gap_report/main.go around lines 168 to 170, the summary log prints
TotalShowsHave, TotalShowsAvail and OverallCompletion but omits TotalMissing;
update the log call to include summary.TotalMissing so the message mirrors the
terminal output (e.g., "Summary: X shows have, Y shows available, Z missing, P%
completion") and adjust the format string and argument order accordingly to
include TotalMissing.

---

- [x] [AI PROMPT - cron/cmd/gap_report/main.go:308]
In cron/cmd/gap_report/main.go around lines 291 to 308, add a warning log when
the loaded ShowsData has an empty Artists map: after initializing shows.Artists
= make(map[string]models.ArtistShowData) (or after unmarshalling), check if
len(shows.Artists) == 0 and emit a concise warning via the existing logger (or
fmt/log package used in the file) indicating that data/shows.json contains no
artists so users can diagnose missing inputs; do not change the function
signature or error behavior, only add the non-fatal log statement.

---

- [x] [AI PROMPT - cron/cmd/monitor/main.go:185]
In cron/cmd/monitor/main.go around lines 176 to 185, the current saveShowsData
writes directly to "data/shows.json" and may fail if the directory doesn't exist
or leave a partially written file; update the function to ensure the directory
exists (use os.MkdirAll for "data"), create a temp file in that directory
(os.CreateTemp), write the marshaled data to the temp file, call file.Sync and
file.Close to flush to disk, then atomically replace the target with
os.Rename(tempPath, "data/shows.json"); handle and log errors at each step and
ensure file permissions are set correctly (0644) on the final file.

---

- [x] [COMMITTABLE SUGGESTION - coderabbitai[bot] - cron/internal/api/handlers/analytics.go:96]
query := &models.AnalyticsQuery{
        ReportType: "artists",
        Timeframe:  timeframe,
        Limit:      limit,
    }

    if ids := c.Query("artist_ids"); ids != "" {
        var artistIDs []int
        for _, s := range strings.Split(ids, ",") {
            if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
                artistIDs = append(artistIDs, id)
            }
        }
        query.ArtistIDs = artistIDs
    }