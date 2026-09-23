Feature: Prometheus metrics
  Bridge was started with --metrics, so /metrics is exposed.

  Scenario: Build info and health are exported
    When "api" sends "GET" to "/metrics"
    Then "api" response status is "200"
    And "api" response header "Content-Type" contains "text/plain"
    And "api" response body contains:
      """
      bridge_build_info{version="
      """
    And "api" response body contains "bridge_storage_up 1"
    And "api" response body contains "# TYPE bridge_http_request_duration_seconds histogram"

  Scenario: Redirects, misses and route changes are counted
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-metrics", "url": "https://example.com/metrics"}
      """
    And "api" sends "GET" to "/e2e-metrics"
    And "api" sends "GET" to "/e2e-metrics-miss"
    When "api" sends "GET" to "/metrics"
    Then "api" response body contains:
      """
      bridge_routes_forwarded_total{key="localhost:18080/e2e-metrics"} 1
      """
    And "api" response body contains:
      """
      bridge_redirects_total{host="localhost:18080"}
      """
    And "api" response body contains:
      """
      bridge_redirect_misses_total{host="localhost:18080"}
      """
    And "api" response body contains:
      """
      bridge_route_changes_total{op="set"}
      """
    And "api" response body contains:
      """
      bridge_http_requests_total{handler="PUT /api/routes",code="202"}
      """
