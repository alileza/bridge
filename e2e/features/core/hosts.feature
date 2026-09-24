Feature: Per-host routes
  The same bridge can serve several hostnames; each has its own short links.

  Scenario: A short link only resolves on the host it was created for
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-host", "url": "https://example.com/localhost-only"}
      """
    When "api" sends "GET" to "/e2e-host"
    Then "api" response header "Location" is "https://example.com/localhost-only"
    When "other_host" sends "GET" to "/e2e-host"
    Then "other_host" response header "Location" is "/"

  Scenario: The same path can point somewhere different on each host
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-shared", "url": "https://example.com/one"}
      """
    And "other_host" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-shared", "url": "https://example.com/two"}
      """
    When "api" sends "GET" to "/e2e-shared"
    Then "api" response header "Location" is "https://example.com/one"
    When "other_host" sends "GET" to "/e2e-shared"
    Then "other_host" response header "Location" is "https://example.com/two"

  Scenario: Listing only shows the current host's routes
    Given "other_host" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-other-only", "url": "https://example.com/other"}
      """
    When "api" sends "GET" to "/api/routes"
    Then "api" response body does not contain "e2e-other-only"
    When "other_host" sends "GET" to "/api/routes"
    Then "other_host" response body contains "127.0.0.1:18080/e2e-other-only"
