Feature: What bridge writes to disk
  The API answers from memory; these scenarios check the routes file and the
  audit log on disk, and that a fresh bridge process loads them (a restart).

  Scenario: A created route is written to the routes file
    When "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-disk", "url": "https://example.com/on-disk"}
      """
    Then "api" response status is "202"
    When "shell" runs:
      """
      jq -e '."localhost:18080/e2e-disk" == "https://example.com/on-disk"' e2e/.data/core.json
      """
    Then "shell" succeeds

  Scenario: An updated route is rewritten in the routes file
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-disk-update", "url": "https://example.com/v1"}
      """
    And "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-disk-update", "url": "https://example.com/v2"}
      """
    When "shell" runs:
      """
      jq -e '."localhost:18080/e2e-disk-update" == "https://example.com/v2"' e2e/.data/core.json
      """
    Then "shell" succeeds

  Scenario: A deleted route is removed from the routes file
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-disk-delete", "url": "https://example.com/gone"}
      """
    And "api" sends "DELETE" to "/api/routes" with json:
      """
      {"key": "localhost:18080/e2e-disk-delete"}
      """
    When "shell" runs:
      """
      jq -e 'has("localhost:18080/e2e-disk-delete") | not' e2e/.data/core.json
      """
    Then "shell" succeeds

  Scenario: Every change is appended to the audit log file
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-disk-audit", "url": "https://example.com/a"}
      """
    And "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-disk-audit", "url": "https://example.com/b"}
      """
    And "api" sends "DELETE" to "/api/routes" with json:
      """
      {"key": "localhost:18080/e2e-disk-audit"}
      """
    When "shell" runs:
      """
      jq -se '
        map(select(.key == "localhost:18080/e2e-disk-audit")) as $e
        | ($e | length) == 3
        and $e[0].action == "create" and $e[0].url == "https://example.com/a" and $e[0].actor == "anonymous"
        and $e[1].action == "update" and $e[1].previous_url == "https://example.com/a" and $e[1].url == "https://example.com/b"
        and $e[2].action == "delete" and $e[2].previous_url == "https://example.com/b"
        and ($e | all(.time | test("^\\d{4}-\\d{2}-\\d{2}T")))
      ' e2e/.data/core.audit.jsonl
      """
    Then "shell" succeeds

  Scenario: A restarted bridge serves the same routes and history
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-restart", "url": "https://example.com/survives"}
      """
    And "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-restart-deleted", "url": "https://example.com/should-stay-gone"}
      """
    And "api" sends "DELETE" to "/api/routes" with json:
      """
      {"key": "localhost:18080/e2e-restart-deleted"}
      """
    # Start a second bridge from a copy of the data files and query it as the same host.
    When "shell" runs:
      """
      set -e
      dir=$(mktemp -d)
      cp e2e/.data/core.json "$dir/core.json"
      cp e2e/.data/core.audit.jsonl "$dir/core.audit.jsonl"
      ./bin/bridge --listen-address 127.0.0.1:18090 --storage "$dir/core" >"$dir/log" 2>&1 &
      pid=$!
      trap 'kill $pid 2>/dev/null' EXIT
      for i in $(seq 50); do curl -sf http://127.0.0.1:18090/healthz >/dev/null && break; sleep 0.1; done
      h='Host: localhost:18080'
      echo "kept=$(curl -s -o /dev/null -w '%{redirect_url}' -H "$h" http://127.0.0.1:18090/e2e-restart)"
      echo "deleted=$(curl -s -o /dev/null -w '%{redirect_url}' -H "$h" http://127.0.0.1:18090/e2e-restart-deleted)"
      echo "history=$(curl -s -H "$h" 'http://127.0.0.1:18090/api/audit?key=localhost:18080/e2e-restart-deleted' | jq -r 'map(.action) | join(",")')"
      """
    Then "shell" succeeds
    And "shell" stdout contains "kept=https://example.com/survives"
    And "shell" stdout contains "deleted=http://127.0.0.1:18090/"
    And "shell" stdout contains "history=delete,create"
