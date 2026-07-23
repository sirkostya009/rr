// This file is a synthetic route set mirroring the deep template surface in
// httx's bench/bench_test.go (18 nested REST templates x 3 API versions x 5
// HTTP methods, plus 5 shallow static routes), expressed as //rr: directives
// so rr can codegen its dispatcher (routes_gen.go) for the comparison in
// bench_test.go against httx, gin and httprouter.
//
// Handlers are no-ops: only routing/dispatch overhead is measured, matching
// how the peer routers register their own bench handlers.
package bench

//go:generate go run github.com/sirkostya009/rr/cmd $GOFILE

type Api struct{}

//rr:route GET /
func (a *Api) Top0Get() {}

//rr:route POST /
func (a *Api) Top0Post() {}

//rr:route PUT /
func (a *Api) Top0Put() {}

//rr:route DELETE /
func (a *Api) Top0Delete() {}

//rr:route PATCH /
func (a *Api) Top0Patch() {}

//rr:route GET /healthz
func (a *Api) Top1Get() {}

//rr:route POST /healthz
func (a *Api) Top1Post() {}

//rr:route PUT /healthz
func (a *Api) Top1Put() {}

//rr:route DELETE /healthz
func (a *Api) Top1Delete() {}

//rr:route PATCH /healthz
func (a *Api) Top1Patch() {}

//rr:route GET /livez
func (a *Api) Top2Get() {}

//rr:route POST /livez
func (a *Api) Top2Post() {}

//rr:route PUT /livez
func (a *Api) Top2Put() {}

//rr:route DELETE /livez
func (a *Api) Top2Delete() {}

//rr:route PATCH /livez
func (a *Api) Top2Patch() {}

//rr:route GET /readyz
func (a *Api) Top3Get() {}

//rr:route POST /readyz
func (a *Api) Top3Post() {}

//rr:route PUT /readyz
func (a *Api) Top3Put() {}

//rr:route DELETE /readyz
func (a *Api) Top3Delete() {}

//rr:route PATCH /readyz
func (a *Api) Top3Patch() {}

//rr:route GET /metrics
func (a *Api) Top4Get() {}

//rr:route POST /metrics
func (a *Api) Top4Post() {}

//rr:route PUT /metrics
func (a *Api) Top4Put() {}

//rr:route DELETE /metrics
func (a *Api) Top4Delete() {}

//rr:route PATCH /metrics
func (a *Api) Top4Patch() {}

//rr:route GET /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Get() {}

//rr:route POST /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Post() {}

//rr:route PUT /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Put() {}

//rr:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Delete() {}

//rr:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V1Patch() {}

//rr:route GET /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Get() {}

//rr:route POST /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Post() {}

//rr:route PUT /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Put() {}

//rr:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Delete() {}

//rr:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V2Patch() {}

//rr:route GET /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Get() {}

//rr:route POST /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Post() {}

//rr:route PUT /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Put() {}

//rr:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Delete() {}

//rr:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}
func (a *Api) D0V3Patch() {}

//rr:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Get() {}

//rr:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Post() {}

//rr:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Put() {}

//rr:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Delete() {}

//rr:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V1Patch() {}

//rr:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Get() {}

//rr:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Post() {}

//rr:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Put() {}

//rr:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Delete() {}

//rr:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V2Patch() {}

//rr:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Get() {}

//rr:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Post() {}

//rr:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Put() {}

//rr:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Delete() {}

//rr:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}
func (a *Api) D1V3Patch() {}

//rr:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Get() {}

//rr:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Post() {}

//rr:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Put() {}

//rr:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Delete() {}

//rr:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V1Patch() {}

//rr:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Get() {}

//rr:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Post() {}

//rr:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Put() {}

//rr:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Delete() {}

//rr:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V2Patch() {}

//rr:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Get() {}

//rr:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Post() {}

//rr:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Put() {}

//rr:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Delete() {}

//rr:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}
func (a *Api) D2V3Patch() {}

//rr:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Get() {}

//rr:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Post() {}

//rr:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Put() {}

//rr:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Delete() {}

//rr:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V1Patch() {}

//rr:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Get() {}

//rr:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Post() {}

//rr:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Put() {}

//rr:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Delete() {}

//rr:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V2Patch() {}

//rr:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Get() {}

//rr:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Post() {}

//rr:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Put() {}

//rr:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Delete() {}

//rr:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
func (a *Api) D3V3Patch() {}

//rr:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Get() {}

//rr:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Post() {}

//rr:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Put() {}

//rr:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Delete() {}

//rr:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V1Patch() {}

//rr:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Get() {}

//rr:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Post() {}

//rr:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Put() {}

//rr:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Delete() {}

//rr:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V2Patch() {}

//rr:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Get() {}

//rr:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Post() {}

//rr:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Put() {}

//rr:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Delete() {}

//rr:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath...}
func (a *Api) D4V3Patch() {}

//rr:route GET /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Get() {}

//rr:route POST /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Post() {}

//rr:route PUT /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Put() {}

//rr:route DELETE /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Delete() {}

//rr:route PATCH /api/v1/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V1Patch() {}

//rr:route GET /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Get() {}

//rr:route POST /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Post() {}

//rr:route PUT /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Put() {}

//rr:route DELETE /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Delete() {}

//rr:route PATCH /api/v2/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V2Patch() {}

//rr:route GET /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Get() {}

//rr:route POST /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Post() {}

//rr:route PUT /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Put() {}

//rr:route DELETE /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Delete() {}

//rr:route PATCH /api/v3/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}
func (a *Api) D5V3Patch() {}

//rr:route GET /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Get() {}

//rr:route POST /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Post() {}

//rr:route PUT /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Put() {}

//rr:route DELETE /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Delete() {}

//rr:route PATCH /api/v1/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V1Patch() {}

//rr:route GET /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Get() {}

//rr:route POST /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Post() {}

//rr:route PUT /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Put() {}

//rr:route DELETE /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Delete() {}

//rr:route PATCH /api/v2/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V2Patch() {}

//rr:route GET /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Get() {}

//rr:route POST /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Post() {}

//rr:route PUT /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Put() {}

//rr:route DELETE /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Delete() {}

//rr:route PATCH /api/v3/organizations/{orgId}/teams/{teamSlug}/members/{userId}
func (a *Api) D6V3Patch() {}

//rr:route GET /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Get() {}

//rr:route POST /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Post() {}

//rr:route PUT /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Put() {}

//rr:route DELETE /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Delete() {}

//rr:route PATCH /api/v1/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V1Patch() {}

//rr:route GET /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Get() {}

//rr:route POST /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Post() {}

//rr:route PUT /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Put() {}

//rr:route DELETE /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Delete() {}

//rr:route PATCH /api/v2/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V2Patch() {}

//rr:route GET /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Get() {}

//rr:route POST /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Post() {}

//rr:route PUT /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Put() {}

//rr:route DELETE /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Delete() {}

//rr:route PATCH /api/v3/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}
func (a *Api) D7V3Patch() {}

//rr:route GET /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Get() {}

//rr:route POST /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Post() {}

//rr:route PUT /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Put() {}

//rr:route DELETE /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Delete() {}

//rr:route PATCH /api/v1/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V1Patch() {}

//rr:route GET /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Get() {}

//rr:route POST /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Post() {}

//rr:route PUT /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Put() {}

//rr:route DELETE /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Delete() {}

//rr:route PATCH /api/v2/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V2Patch() {}

//rr:route GET /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Get() {}

//rr:route POST /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Post() {}

//rr:route PUT /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Put() {}

//rr:route DELETE /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Delete() {}

//rr:route PATCH /api/v3/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
func (a *Api) D8V3Patch() {}

//rr:route GET /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Get() {}

//rr:route POST /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Post() {}

//rr:route PUT /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Put() {}

//rr:route DELETE /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Delete() {}

//rr:route PATCH /api/v1/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V1Patch() {}

//rr:route GET /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Get() {}

//rr:route POST /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Post() {}

//rr:route PUT /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Put() {}

//rr:route DELETE /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Delete() {}

//rr:route PATCH /api/v2/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V2Patch() {}

//rr:route GET /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Get() {}

//rr:route POST /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Post() {}

//rr:route PUT /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Put() {}

//rr:route DELETE /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Delete() {}

//rr:route PATCH /api/v3/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}
func (a *Api) D9V3Patch() {}

//rr:route GET /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Get() {}

//rr:route POST /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Post() {}

//rr:route PUT /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Put() {}

//rr:route DELETE /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Delete() {}

//rr:route PATCH /api/v1/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V1Patch() {}

//rr:route GET /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Get() {}

//rr:route POST /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Post() {}

//rr:route PUT /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Put() {}

//rr:route DELETE /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Delete() {}

//rr:route PATCH /api/v2/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V2Patch() {}

//rr:route GET /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Get() {}

//rr:route POST /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Post() {}

//rr:route PUT /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Put() {}

//rr:route DELETE /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Delete() {}

//rr:route PATCH /api/v3/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}
func (a *Api) D10V3Patch() {}

//rr:route GET /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Get() {}

//rr:route POST /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Post() {}

//rr:route PUT /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Put() {}

//rr:route DELETE /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Delete() {}

//rr:route PATCH /api/v1/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V1Patch() {}

//rr:route GET /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Get() {}

//rr:route POST /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Post() {}

//rr:route PUT /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Put() {}

//rr:route DELETE /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Delete() {}

//rr:route PATCH /api/v2/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V2Patch() {}

//rr:route GET /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Get() {}

//rr:route POST /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Post() {}

//rr:route PUT /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Put() {}

//rr:route DELETE /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Delete() {}

//rr:route PATCH /api/v3/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}
func (a *Api) D11V3Patch() {}

//rr:route GET /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Get() {}

//rr:route POST /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Post() {}

//rr:route PUT /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Put() {}

//rr:route DELETE /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Delete() {}

//rr:route PATCH /api/v1/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V1Patch() {}

//rr:route GET /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Get() {}

//rr:route POST /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Post() {}

//rr:route PUT /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Put() {}

//rr:route DELETE /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Delete() {}

//rr:route PATCH /api/v2/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V2Patch() {}

//rr:route GET /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Get() {}

//rr:route POST /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Post() {}

//rr:route PUT /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Put() {}

//rr:route DELETE /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Delete() {}

//rr:route PATCH /api/v3/datasets/{datasetId}/tables/{tableId}/columns/{columnId}
func (a *Api) D12V3Patch() {}

//rr:route GET /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Get() {}

//rr:route POST /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Post() {}

//rr:route PUT /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Put() {}

//rr:route DELETE /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Delete() {}

//rr:route PATCH /api/v1/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V1Patch() {}

//rr:route GET /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Get() {}

//rr:route POST /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Post() {}

//rr:route PUT /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Put() {}

//rr:route DELETE /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Delete() {}

//rr:route PATCH /api/v2/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V2Patch() {}

//rr:route GET /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Get() {}

//rr:route POST /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Post() {}

//rr:route PUT /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Put() {}

//rr:route DELETE /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Delete() {}

//rr:route PATCH /api/v3/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}
func (a *Api) D13V3Patch() {}

//rr:route GET /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Get() {}

//rr:route POST /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Post() {}

//rr:route PUT /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Put() {}

//rr:route DELETE /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Delete() {}

//rr:route PATCH /api/v1/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V1Patch() {}

//rr:route GET /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Get() {}

//rr:route POST /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Post() {}

//rr:route PUT /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Put() {}

//rr:route DELETE /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Delete() {}

//rr:route PATCH /api/v2/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V2Patch() {}

//rr:route GET /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Get() {}

//rr:route POST /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Post() {}

//rr:route PUT /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Put() {}

//rr:route DELETE /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Delete() {}

//rr:route PATCH /api/v3/webhooks/{whId}/deliveries/{deliveryId}
func (a *Api) D14V3Patch() {}

//rr:route GET /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Get() {}

//rr:route POST /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Post() {}

//rr:route PUT /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Put() {}

//rr:route DELETE /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Delete() {}

//rr:route PATCH /api/v1/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V1Patch() {}

//rr:route GET /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Get() {}

//rr:route POST /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Post() {}

//rr:route PUT /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Put() {}

//rr:route DELETE /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Delete() {}

//rr:route PATCH /api/v2/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V2Patch() {}

//rr:route GET /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Get() {}

//rr:route POST /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Post() {}

//rr:route PUT /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Put() {}

//rr:route DELETE /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Delete() {}

//rr:route PATCH /api/v3/integrations/{provSlug}/connections/{connId}/syncs/{syncId}
func (a *Api) D15V3Patch() {}

//rr:route GET /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Get(orderId, lineNo int) {}

//rr:route POST /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Post(orderId, lineNo int) {}

//rr:route PUT /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Put(orderId, lineNo int) {}

//rr:route DELETE /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Delete(orderId, lineNo int) {}

//rr:route PATCH /api/v1/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V1Patch(orderId, lineNo int) {}

//rr:route GET /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Get(orderId, lineNo int) {}

//rr:route POST /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Post(orderId, lineNo int) {}

//rr:route PUT /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Put(orderId, lineNo int) {}

//rr:route DELETE /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Delete(orderId, lineNo int) {}

//rr:route PATCH /api/v2/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V2Patch(orderId, lineNo int) {}

//rr:route GET /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Get(orderId, lineNo int) {}

//rr:route POST /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Post(orderId, lineNo int) {}

//rr:route PUT /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Put(orderId, lineNo int) {}

//rr:route DELETE /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Delete(orderId, lineNo int) {}

//rr:route PATCH /api/v3/orders/{orderId}/lines/{lineNo}
func (a *Api) D16V3Patch(orderId, lineNo int) {}

//rr:route GET /api/v1/sessions/{sessionId}
func (a *Api) D17V1Get() {}

//rr:route POST /api/v1/sessions/{sessionId}
func (a *Api) D17V1Post() {}

//rr:route PUT /api/v1/sessions/{sessionId}
func (a *Api) D17V1Put() {}

//rr:route DELETE /api/v1/sessions/{sessionId}
func (a *Api) D17V1Delete() {}

//rr:route PATCH /api/v1/sessions/{sessionId}
func (a *Api) D17V1Patch() {}

//rr:route GET /api/v2/sessions/{sessionId}
func (a *Api) D17V2Get() {}

//rr:route POST /api/v2/sessions/{sessionId}
func (a *Api) D17V2Post() {}

//rr:route PUT /api/v2/sessions/{sessionId}
func (a *Api) D17V2Put() {}

//rr:route DELETE /api/v2/sessions/{sessionId}
func (a *Api) D17V2Delete() {}

//rr:route PATCH /api/v2/sessions/{sessionId}
func (a *Api) D17V2Patch() {}

//rr:route GET /api/v3/sessions/{sessionId}
func (a *Api) D17V3Get() {}

//rr:route POST /api/v3/sessions/{sessionId}
func (a *Api) D17V3Post() {}

//rr:route PUT /api/v3/sessions/{sessionId}
func (a *Api) D17V3Put() {}

//rr:route DELETE /api/v3/sessions/{sessionId}
func (a *Api) D17V3Delete() {}

//rr:route PATCH /api/v3/sessions/{sessionId}
func (a *Api) D17V3Patch() {}
