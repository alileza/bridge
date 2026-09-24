Feature: GitHub login
  With GitHub login enabled the portal needs a session, short links stay public.

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

  Scenario: The portal and API require login
    When "api" sends "GET" to "/"
    Then "api" response status is "302"
    And "api" response header "Location" is "/auth/login"
    When "api" sends "GET" to "/api/routes"
    Then "api" response status is "401"
    And "api" response json "error" is "login required"
    When "api" sends "PUT" to "/api/routes" with json:
      """
      {"key": "/e2e-anon", "url": "https://example.com"}
      """
    Then "api" response status is "401"
    When "api" sends "GET" to "/api/audit"
    Then "api" response status is "401"

  Scenario: Health and auth status stay public
    When "api" sends "GET" to "/healthz"
    Then "api" response status is "200"
    When "api" sends "GET" to "/api/me"
    Then "api" response json "auth_enabled" is "true"
    And "api" response json "authenticated" is "false"

  Scenario: Login redirects to GitHub with the right scopes
    When "api" sends "GET" to "/auth/login"
    Then "api" response status is "302"
    And "api" response header "Location" contains "http://127.0.0.1:19999/login/oauth/authorize?"
    And "api" response header "Location" contains "client_id=e2e-client"
    And "api" response header "Location" contains "scope=read%3Auser+user%3Aemail+read%3Aorg"
    And "api" response header "Location" contains "redirect_uri=http%3A%2F%2Flocalhost%3A18081%2Fauth%2Fcallback"
    And "api" response cookie "bridge_oauth_state" exists

  Scenario: A member of an allowed org logs in
    Given "github" stub "GET" "/api/v3/user/orgs" returns "200" with json:
      """
      [{"login": "other"}, {"login": "ACME"}]
      """
    And "api" cookie "bridge_oauth_state" is "e2e-state"
    When "api" sends "GET" to "/auth/callback?code=e2e-code&state=e2e-state"
    Then "api" response status is "302"
    And "api" response header "Location" is "/"
    And "api" response cookie "bridge_session" exists
    And "github" received request with body containing "client_secret=e2e-secret"
    And "github" received request with header "Authorization" containing "Bearer gho_e2e"

  Scenario: A user outside the allowed orgs is refused
    Given "github" stub "GET" "/api/v3/user/orgs" returns "200" with json:
      """
      [{"login": "evil-corp"}]
      """
    And "api" cookie "bridge_oauth_state" is "e2e-state"
    When "api" sends "GET" to "/auth/callback?code=e2e-code&state=e2e-state"
    Then "api" response status is "403"
    And "api" response body contains "@alice is not allowed to use this bridge"

  Scenario: A forged OAuth state is rejected before calling GitHub
    Given "api" cookie "bridge_oauth_state" is "real-state"
    When "api" sends "GET" to "/auth/callback?code=e2e-code&state=attacker-state"
    Then "api" response status is "400"
    And "github" did not receive "POST" "/login/oauth/access_token"

  Scenario: A tampered session is not accepted
    Given "api" cookie "bridge_session" is "eyJsIjoiYWRtaW4iLCJ4Ijo5OTk5OTk5OTk5fQ.forged"
    When "api" sends "GET" to "/api/routes"
    Then "api" response status is "401"

  Scenario: The session cookie is locked down
    Given "github" stub "GET" "/api/v3/user/orgs" returns "200" with json:
      """
      [{"login": "acme"}]
      """
    And "api" cookie "bridge_oauth_state" is "e2e-state"
    When "api" sends "GET" to "/auth/callback?code=e2e-code&state=e2e-state"
    Then "api" response header "Set-Cookie" contains "bridge_oauth_state=;"
    When "shell" runs:
      """
      curl -s -o /dev/null -D - --cookie 'bridge_oauth_state=e2e-state' \
        'http://localhost:18081/auth/callback?code=e2e-code&state=e2e-state' \
        | grep -i '^set-cookie: bridge_session='
      """
    Then "shell" succeeds
    And "shell" stdout contains "HttpOnly"
    And "shell" stdout contains "SameSite=Lax"
    And "shell" stdout contains "Path=/"
