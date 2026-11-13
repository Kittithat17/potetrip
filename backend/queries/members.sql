-- name: CreateMember :one
INSERT INTO members (trip_id, name, promptpay)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMembers :many
SELECT * FROM members
WHERE trip_id = $1
ORDER BY created_at, id;

-- name: GetMember :one
SELECT * FROM members WHERE id = $1;

-- name: UpdateMemberPromptPay :one
UPDATE members SET promptpay = $3
WHERE id = $1 AND trip_id = $2
RETURNING *;

-- name: ListMembersWithLink :many
-- linked: some signed-in account already acts as this member, so nobody
-- else may claim to be them.
SELECT m.id, m.name,
       EXISTS (SELECT 1 FROM devices d
               WHERE d.member_id = m.id AND d.status = 'approved' AND d.user_id IS NOT NULL) AS linked
FROM members m
WHERE m.trip_id = $1
ORDER BY m.created_at, m.id;

-- name: MemberLinked :one
SELECT EXISTS (SELECT 1 FROM devices d
               WHERE d.member_id = $1 AND d.status = 'approved' AND d.user_id IS NOT NULL);
