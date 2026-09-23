Feature: Short links
  Bridge stores short paths per host and redirects them to their destination.

  Scenario: Create a short link and follow it
    When "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-create", "url": "https://github.com/alileza/bridge"}
      """
    Then "api" response status is "202"
    When "api" sends "GET" to "/e2e-create"
    Then "api" response status is "302"
    And "api" response header "Location" is "https://github.com/alileza/bridge"

  Scenario: A key without a leading slash is normalised
    When "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "e2e-noslash", "url": "https://example.com/noslash"}
      """
    Then "api" response status is "202"
    When "api" sends "GET" to "/e2e-noslash"
    Then "api" response header "Location" is "https://example.com/noslash"

  Scenario: Update an existing short link
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-update", "url": "https://example.com/v1"}
      """
    When "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-update", "url": "https://example.com/v2"}
      """
    Then "api" response status is "202"
    When "api" sends "GET" to "/e2e-update"
    Then "api" response header "Location" is "https://example.com/v2"

  Scenario: Delete a short link
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-delete", "url": "https://example.com/gone"}
      """
    When "api" sends "DELETE" to "/api/routes" with json:
      """
      {"key": "localhost:18080/e2e-delete"}
      """
    Then "api" response status is "200"
    When "api" sends "GET" to "/e2e-delete"
    Then "api" response status is "302"
    And "api" response header "Location" is "/"

  Scenario: List routes for the current host
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-list", "url": "https://example.com/list"}
      """
    When "api" sends "GET" to "/api/routes"
    Then "api" response status is "200"
    And "api" response header "Content-Type" contains "application/json"
    And "api" response body contains:
      """
      "key":"localhost:18080/e2e-list","url":"https://example.com/list"
      """

  Scenario Outline: Reject invalid routes
    When "api" sends "PUT" to "/api/routes" with body:
      """
      <body>
      """
    Then "api" response status is "400"
    And "api" response json "error" matches pattern "<error>"

    Examples:
      | body                                        | error                   |
      | {"key": "", "url": "https://example.com"}   | empty key               |
      | {"key": "/e2e-bad", "url": ""}              | empty url               |
      | {"key": "/e2e-bad", "url": "not a url"}     | invalid destination URL |
      | not json                                    | invalid character       |
