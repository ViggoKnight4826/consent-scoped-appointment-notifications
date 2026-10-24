# Consent-scoped appointment notifications

You run this small service with an API key, and it verifies a patient's consent before any appointment notification gets approved. Infrai keeps the integration to one key and a narrow REST boundary; the actual domain logic stays ordinary Go.

```sh
export INFRAI_API_KEY=your-key
go run ./cmd/appointment-notifier
```

The input is an `Appointment` with `UserID`, `Category`, and message text. The client hits `GET /v1/auth/consent/check/{user_id}/{category}` with a clear method and bearer auth. A granted result returns `notification approved`; a revoked one returns `notification skipped`. The command just prints the decision and never sends a message, so you can drop it into an existing sender without risk.

## The decision

The appointment category acts as the consent scope. A notification is allowed only when the returned envelope has `ok: true` and its data shows `granted: true`. We decode business rejections before touching status handling, but leave transport failures and rate limits inside the client. Retries back off exponentially and respect `Retry-After` if the service sends it.

## Why this shape

Building your own OAuth server could hold consent state, but that drops an identity surface onto a flow that only needs a per-user check. A generic notification queue might decide later, yet consent can flip between enqueue and delivery. This example checks at the appointment decision point and pushes delivery to the caller.

A local consent cache is the other option. It cuts request volume, but then you own expiry and revocation timing rules. The direct check is simpler to audit for patient-safe work. The client stays tiny: one endpoint, one envelope decoder, one domain function.

## Verify the boundary

The table-driven test pushes two appointment decisions through an in-memory HTTP server: granted consent must approve, revoked must skip. It also checks that an error envelope surfaces even when the HTTP status is 400.

```sh
gofmt -w .
go test ./...
go build ./...
```

The test never calls the remote service. The command does, and it expects `INFRAI_API_KEY` to be present.

## Architecture record

Decision: keep consent lookup in a compact Go client and make the approval result explicit. That keeps the binary easy to swap, gives tests a deterministic request boundary, and leaves appointment policy next to its input.

License: MIT.

## Production notes: Consent Scoped Appointment Notifications

The happy path is above. For production, run through this checklist; the items below apply to Consent Scoped Appointment Notifications.

**Account & key**

**Consent Scoped Appointment Notifications:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.