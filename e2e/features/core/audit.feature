Feature: Audit log without GitHub login
  Changes are still recorded, attributed to "anonymous".

  Scenario: Create, update and delete are recorded newest first
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-audit", "url": "https://example.com/a"}
      """
    And "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-audit", "url": "https://example.com/b"}
      """
    And "api" sends "DELETE" to "/api/routes" with json:
      """
      {"key": "localhost:18080/e2e-audit"}
      """
    Given "api" query param "key" is "localhost:18080/e2e-audit"
    When "api" sends "GET" to "/api/audit"
    Then "api" response status is "200"
    And "api" response json "[0].action" is "delete"
    And "api" response json "[0].actor" is "anonymous"
    And "api" response json "[0].previous_url" is "https://example.com/b"
    And "api" response json "[1].action" is "update"
    And "api" response json "[1].url" is "https://example.com/b"
    And "api" response json "[1].previous_url" is "https://example.com/a"
    And "api" response json "[2].action" is "create"
    And "api" response json "[2].url" is "https://example.com/a"
    And "api" response json "[0].time" is iso-timestamp
    And "api" response json "[0].actor_emails" does not exist

  Scenario: Saving the same URL again is not recorded
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-noop", "url": "https://example.com/same"}
      """
    And "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-noop", "url": "https://example.com/same"}
      """
    Given "api" query param "key" is "localhost:18080/e2e-noop"
    When "api" sends "GET" to "/api/audit"
    Then "api" response json "[0].action" is "create"
    And "api" response json "[1]" does not exist

  Scenario: Routes show who changed them last
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-attributed", "url": "https://example.com/who"}
      """
    When "api" sends "GET" to "/api/routes"
    Then "api" response body contains:
      """
      "key":"localhost:18080/e2e-attributed","url":"https://example.com/who","updated_by":"anonymous"
      """
