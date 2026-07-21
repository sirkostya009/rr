// This file is a synthetic route set mirroring the deep template surface in
// httx's bench/bench_test.go (18 nested REST templates x 3 API versions x 5
// HTTP methods, plus 5 shallow static routes), expressed as //api: directives
// so rr can codegen its dispatcher (routes_gen.go) for the comparison in
// bench_test.go against httx, gin and httprouter.
//
// Handlers are no-ops: only routing/dispatch overhead is measured, matching
// how the peer routers register their own bench handlers.
package bench

//go:generate go run github.com/sirkostya009/rr/cmd $GOFILE

type Api struct{}

//api:route GET /
func (a *Api) Top0Get() {}

//api:route POST /
func (a *Api) Top0Post() {}

//api:route PUT /
func (a *Api) Top0Put() {}

//api:route DELETE /
func (a *Api) Top0Delete() {}

//api:route PATCH /
func (a *Api) Top0Patch() {}

//api:route GET /healthz
func (a *Api) Top1Get() {}

//api:route POST /healthz
func (a *Api) Top1Post() {}

//api:route PUT /healthz
func (a *Api) Top1Put() {}

//api:route DELETE /healthz
func (a *Api) Top1Delete() {}

//api:route PATCH /healthz
func (a *Api) Top1Patch() {}

//api:route GET /livez
func (a *Api) Top2Get() {}

//api:route POST /livez
func (a *Api) Top2Post() {}

//api:route PUT /livez
func (a *Api) Top2Put() {}

//api:route DELETE /livez
func (a *Api) Top2Delete() {}

//api:route PATCH /livez
func (a *Api) Top2Patch() {}

//api:route GET /readyz
func (a *Api) Top3Get() {}

//api:route POST /readyz
func (a *Api) Top3Post() {}

//api:route PUT /readyz
func (a *Api) Top3Put() {}

//api:route DELETE /readyz
func (a *Api) Top3Delete() {}

//api:route PATCH /readyz
func (a *Api) Top3Patch() {}

//api:route GET /metrics
func (a *Api) Top4Get() {}

//api:route POST /metrics
func (a *Api) Top4Post() {}

//api:route PUT /metrics
func (a *Api) Top4Put() {}

//api:route DELETE /metrics
func (a *Api) Top4Delete() {}

//api:route PATCH /metrics
func (a *Api) Top4Patch() {}

//api:route GET /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Get() {}

//api:route POST /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Post() {}

//api:route PUT /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Put() {}

//api:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Delete() {}

//api:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Patch() {}

//api:route GET /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Get() {}

//api:route POST /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Post() {}

//api:route PUT /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Put() {}

//api:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Delete() {}

//api:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Patch() {}

//api:route GET /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Get() {}

//api:route POST /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Post() {}

//api:route PUT /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Put() {}

//api:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Delete() {}

//api:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Patch() {}

//api:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Get() {}

//api:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Post() {}

//api:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Put() {}

//api:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Delete() {}

//api:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Patch() {}

//api:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Get() {}

//api:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Post() {}

//api:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Put() {}

//api:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Delete() {}

//api:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Patch() {}

//api:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Get() {}

//api:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Post() {}

//api:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Put() {}

//api:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Delete() {}

//api:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Patch() {}

//api:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Get() {}

//api:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Post() {}

//api:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Put() {}

//api:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Delete() {}

//api:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Patch() {}

//api:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Get() {}

//api:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Post() {}

//api:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Put() {}

//api:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Delete() {}

//api:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Patch() {}

//api:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Get() {}

//api:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Post() {}

//api:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Put() {}

//api:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Delete() {}

//api:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Patch() {}

//api:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Get() {}

//api:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Post() {}

//api:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Put() {}

//api:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Delete() {}

//api:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Patch() {}

//api:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Get() {}

//api:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Post() {}

//api:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Put() {}

//api:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Delete() {}

//api:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Patch() {}

//api:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Get() {}

//api:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Post() {}

//api:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Put() {}

//api:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Delete() {}

//api:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Patch() {}

//api:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Get() {}

//api:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Post() {}

//api:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Put() {}

//api:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Delete() {}

//api:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Patch() {}

//api:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Get() {}

//api:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Post() {}

//api:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Put() {}

//api:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Delete() {}

//api:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Patch() {}

//api:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Get() {}

//api:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Post() {}

//api:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Put() {}

//api:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Delete() {}

//api:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Patch() {}

//api:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Get() {}

//api:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Post() {}

//api:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Put() {}

//api:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Delete() {}

//api:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Patch() {}

//api:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Get() {}

//api:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Post() {}

//api:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Put() {}

//api:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Delete() {}

//api:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Patch() {}

//api:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Get() {}

//api:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Post() {}

//api:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Put() {}

//api:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Delete() {}

//api:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Patch() {}

//api:route GET /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Get() {}

//api:route POST /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Post() {}

//api:route PUT /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Put() {}

//api:route DELETE /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Delete() {}

//api:route PATCH /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Patch() {}

//api:route GET /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Get() {}

//api:route POST /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Post() {}

//api:route PUT /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Put() {}

//api:route DELETE /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Delete() {}

//api:route PATCH /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Patch() {}

//api:route GET /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Get() {}

//api:route POST /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Post() {}

//api:route PUT /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Put() {}

//api:route DELETE /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Delete() {}

//api:route PATCH /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Patch() {}

//api:route GET /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Get() {}

//api:route POST /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Post() {}

//api:route PUT /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Put() {}

//api:route DELETE /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Delete() {}

//api:route PATCH /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Patch() {}

//api:route GET /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Get() {}

//api:route POST /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Post() {}

//api:route PUT /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Put() {}

//api:route DELETE /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Delete() {}

//api:route PATCH /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Patch() {}

//api:route GET /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Get() {}

//api:route POST /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Post() {}

//api:route PUT /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Put() {}

//api:route DELETE /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Delete() {}

//api:route PATCH /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Patch() {}

//api:route GET /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Get() {}

//api:route POST /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Post() {}

//api:route PUT /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Put() {}

//api:route DELETE /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Delete() {}

//api:route PATCH /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Patch() {}

//api:route GET /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Get() {}

//api:route POST /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Post() {}

//api:route PUT /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Put() {}

//api:route DELETE /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Delete() {}

//api:route PATCH /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Patch() {}

//api:route GET /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Get() {}

//api:route POST /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Post() {}

//api:route PUT /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Put() {}

//api:route DELETE /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Delete() {}

//api:route PATCH /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Patch() {}

//api:route GET /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Get() {}

//api:route POST /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Post() {}

//api:route PUT /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Put() {}

//api:route DELETE /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Delete() {}

//api:route PATCH /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Patch() {}

//api:route GET /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Get() {}

//api:route POST /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Post() {}

//api:route PUT /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Put() {}

//api:route DELETE /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Delete() {}

//api:route PATCH /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Patch() {}

//api:route GET /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Get() {}

//api:route POST /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Post() {}

//api:route PUT /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Put() {}

//api:route DELETE /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Delete() {}

//api:route PATCH /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Patch() {}

//api:route GET /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Get() {}

//api:route POST /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Post() {}

//api:route PUT /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Put() {}

//api:route DELETE /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Delete() {}

//api:route PATCH /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Patch() {}

//api:route GET /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Get() {}

//api:route POST /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Post() {}

//api:route PUT /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Put() {}

//api:route DELETE /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Delete() {}

//api:route PATCH /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Patch() {}

//api:route GET /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Get() {}

//api:route POST /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Post() {}

//api:route PUT /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Put() {}

//api:route DELETE /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Delete() {}

//api:route PATCH /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Patch() {}

//api:route GET /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Get() {}

//api:route POST /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Post() {}

//api:route PUT /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Put() {}

//api:route DELETE /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Delete() {}

//api:route PATCH /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Patch() {}

//api:route GET /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Get() {}

//api:route POST /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Post() {}

//api:route PUT /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Put() {}

//api:route DELETE /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Delete() {}

//api:route PATCH /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Patch() {}

//api:route GET /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Get() {}

//api:route POST /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Post() {}

//api:route PUT /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Put() {}

//api:route DELETE /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Delete() {}

//api:route PATCH /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Patch() {}

//api:route GET /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Get() {}

//api:route POST /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Post() {}

//api:route PUT /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Put() {}

//api:route DELETE /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Delete() {}

//api:route PATCH /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Patch() {}

//api:route GET /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Get() {}

//api:route POST /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Post() {}

//api:route PUT /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Put() {}

//api:route DELETE /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Delete() {}

//api:route PATCH /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Patch() {}

//api:route GET /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Get() {}

//api:route POST /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Post() {}

//api:route PUT /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Put() {}

//api:route DELETE /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Delete() {}

//api:route PATCH /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Patch() {}

//api:route GET /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Get() {}

//api:route POST /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Post() {}

//api:route PUT /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Put() {}

//api:route DELETE /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Delete() {}

//api:route PATCH /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Patch() {}

//api:route GET /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Get() {}

//api:route POST /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Post() {}

//api:route PUT /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Put() {}

//api:route DELETE /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Delete() {}

//api:route PATCH /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Patch() {}

//api:route GET /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Get() {}

//api:route POST /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Post() {}

//api:route PUT /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Put() {}

//api:route DELETE /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Delete() {}

//api:route PATCH /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Patch() {}

//api:route GET /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Get() {}

//api:route POST /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Post() {}

//api:route PUT /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Put() {}

//api:route DELETE /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Delete() {}

//api:route PATCH /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Patch() {}

//api:route GET /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Get() {}

//api:route POST /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Post() {}

//api:route PUT /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Put() {}

//api:route DELETE /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Delete() {}

//api:route PATCH /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Patch() {}

//api:route GET /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Get() {}

//api:route POST /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Post() {}

//api:route PUT /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Put() {}

//api:route DELETE /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Delete() {}

//api:route PATCH /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Patch() {}

//api:route GET /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Get() {}

//api:route POST /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Post() {}

//api:route PUT /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Put() {}

//api:route DELETE /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Delete() {}

//api:route PATCH /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Patch() {}

//api:route GET /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Get() {}

//api:route POST /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Post() {}

//api:route PUT /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Put() {}

//api:route DELETE /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Delete() {}

//api:route PATCH /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Patch() {}

//api:route GET /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Get() {}

//api:route POST /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Post() {}

//api:route PUT /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Put() {}

//api:route DELETE /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Delete() {}

//api:route PATCH /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Patch() {}

//api:route GET /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Get(orderId, lineNo int) {}

//api:route POST /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Post(orderId, lineNo int) {}

//api:route PUT /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Put(orderId, lineNo int) {}

//api:route DELETE /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Delete(orderId, lineNo int) {}

//api:route PATCH /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Patch(orderId, lineNo int) {}

//api:route GET /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Get(orderId, lineNo int) {}

//api:route POST /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Post(orderId, lineNo int) {}

//api:route PUT /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Put(orderId, lineNo int) {}

//api:route DELETE /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Delete(orderId, lineNo int) {}

//api:route PATCH /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Patch(orderId, lineNo int) {}

//api:route GET /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Get(orderId, lineNo int) {}

//api:route POST /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Post(orderId, lineNo int) {}

//api:route PUT /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Put(orderId, lineNo int) {}

//api:route DELETE /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Delete(orderId, lineNo int) {}

//api:route PATCH /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Patch(orderId, lineNo int) {}

//api:route GET /api/v1/sessions/{sessionId}
func (a *Api) D17V1Get() {}

//api:route POST /api/v1/sessions/{sessionId}
func (a *Api) D17V1Post() {}

//api:route PUT /api/v1/sessions/{sessionId}
func (a *Api) D17V1Put() {}

//api:route DELETE /api/v1/sessions/{sessionId}
func (a *Api) D17V1Delete() {}

//api:route PATCH /api/v1/sessions/{sessionId}
func (a *Api) D17V1Patch() {}

//api:route GET /api/v2/sessions/{sessionId}
func (a *Api) D17V2Get() {}

//api:route POST /api/v2/sessions/{sessionId}
func (a *Api) D17V2Post() {}

//api:route PUT /api/v2/sessions/{sessionId}
func (a *Api) D17V2Put() {}

//api:route DELETE /api/v2/sessions/{sessionId}
func (a *Api) D17V2Delete() {}

//api:route PATCH /api/v2/sessions/{sessionId}
func (a *Api) D17V2Patch() {}

//api:route GET /api/v3/sessions/{sessionId}
func (a *Api) D17V3Get() {}

//api:route POST /api/v3/sessions/{sessionId}
func (a *Api) D17V3Post() {}

//api:route PUT /api/v3/sessions/{sessionId}
func (a *Api) D17V3Put() {}

//api:route DELETE /api/v3/sessions/{sessionId}
func (a *Api) D17V3Delete() {}

//api:route PATCH /api/v3/sessions/{sessionId}
func (a *Api) D17V3Patch() {}
