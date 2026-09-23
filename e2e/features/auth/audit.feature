Feature: Audit log with GitHub login
  Changes are attributed to the logged-in GitHub user and their verified emails.

  Background:
    Given "github" stub "POST" "/login/oauth/access_token" returns "200" with json:
      """
      {"access_token": "gho_e2e"}
      """
    And "github" stub "GET" "/api/v3/user" returns "200" with json:
      """
      {"login": "alice", "email": null}
      """
    And "github" stub "GET" "/api/v3/user/emails" returns "200" with json:
      """
      [
        {"email": "alice@personal.dev", "primary": false, "verified": true},
        {"email": "alice@unverified.dev", "primary": false, "verified": false},
        {"email": "alice@acme.com", "primary": true, "verified": true}
      ]
      """
    And "github" stub "GET" "/api/v3/user/orgs" returns "200" with json:
      """
      [{"login": "acme"}]
      """
    And "api" cookie "bridge_oauth_state" is "e2e-state"
    And "api" sends "GET" to "/auth/callback?code=e2e-code&state=e2e-state"
    And "api" response cookie "bridge_session" saved as "{{session}}"
    And "api" cookie "bridge_session" is "{{session}}"

  Scenario: The session identifies the user
    When "api" sends "GET" to "/api/me"
    Then "api" response json "authenticated" is "true"
    And "api" response json "login" is "alice"
    And "api" response json "emails[0]" is "alice@acme.com"
    And "api" response json "emails[1]" is "alice@personal.dev"
    And "api" response json "emails[2]" does not exist

  Scenario: Changes are attributed to the user with their verified emails
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-alice", "url": "https://example.com/v1"}
      """
    And "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-alice", "url": "https://example.com/v2"}
      """
    Given "api" query param "key" is "localhost:18081/e2e-alice"
    When "api" sends "GET" to "/api/audit"
    Then "api" response status is "200"
    And "api" response json "[0].actor" is "alice"
    And "api" response json "[0].action" is "update"
    And "api" response json "[0].actor_emails[0]" is "alice@acme.com"
    And "api" response json "[0].actor_emails[1]" is "alice@personal.dev"
    And "api" response json "[1].actor" is "alice"
    And "api" response json "[1].action" is "create"

  Scenario: Short links stay public
    Given "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-public", "url": "https://example.com/public"}
      """
    And "api" cookie "bridge_session" is "logged-out"
    When "api" sends "GET" to "/e2e-public"
    Then "api" response status is "302"
    And "api" response header "Location" is "https://example.com/public"

  Scenario: Logging out ends the session
    When "api" sends "POST" to "/auth/logout"
    Then "api" response status is "204"
    And "api" response cookie "bridge_session" is ""
