-- name: UpsertPrice :exec
INSERT OR REPLACE INTO prices (apt_name, lawd_cd, month, area_int, area, median, count)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: InsertPriceIgnore :exec
INSERT OR IGNORE INTO prices (apt_name, lawd_cd, month, area_int, area, median, count)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListPricesByApt :many
SELECT month, area_int, area, median, count
FROM prices
WHERE apt_name = ? AND lawd_cd = ? AND area_int = ?
ORDER BY month DESC;

-- name: GetCachedMonths :many
SELECT month
FROM prices
WHERE apt_name = ? AND lawd_cd = ? AND area_int = ?;
