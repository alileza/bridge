Feature: Portal and health endpoints

  Scenario: The portal UI is served
    When "api" sends "GET" to "/"
    Then "api" response status is "200"
    And "api" response header "Content-Type" contains "text/html"
    And "api" response body contains:
      """
      <div id="root"></div>
      """
    And "api" response body contains "/favicon.png"

  Scenario Outline: Static assets are served with the right content type
    When "api" sends "GET" to "<path>"
    Then "api" response status is "200"
    And "api" response header "Content-Type" contains "image/png"

    Examples:
      | path                  |
      | /favicon.ico          |
      | /favicon.png          |
      | /apple-touch-icon.png |
      | /bridge.png           |

  Scenario: Unknown paths go back to the portal
    When "api" sends "GET" to "/e2e-definitely-not-a-link"
    Then "api" response status is "302"
    And "api" response header "Location" is "/"

  Scenario: Health check
    When "api" sends "GET" to "/healthz"
    Then "api" response status is "200"
    And "api" response json "status" is "ok"

  Scenario: Auth status when GitHub login is disabled
    When "api" sends "GET" to "/api/me"
    Then "api" response status is "200"
    And "api" response json "auth_enabled" is "false"
    And "api" response json "authenticated" is "false"

  Scenario: QR code for a URL
    Given "api" query param "url" is "https://example.com"
    When "api" sends "GET" to "/api/routes/barcode"
    Then "api" response status is "200"
    And "api" response header "Content-Type" is "image/png"

  Scenario: QR code requires a URL
    When "api" sends "GET" to "/api/routes/barcode"
    Then "api" response status is "400"
    And "api" response json "error" is "empty url"
