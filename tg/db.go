package tg

import "github.com/jmoiron/sqlx"

func rollbackHash(db *sqlx.DB, id int64) {
	db.Exec("update hashes set used = 0 where id = ?", id)
}
func removeHash(db *sqlx.DB, id int64) {
	db.Exec("delete from hashes where id = ?", id)
}
