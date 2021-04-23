package mbtiles

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// MBTiles todo
type MBTiles struct {
	db             *sql.DB
	tileInsertStmt *sql.Stmt
	// 	Ncols, Nrows     uint
	// 	Xcenter, Ycenter *float64
	// 	Xcorner, Ycorner *float64
	// 	CellSize         float64
	// 	NoDataValue      float64
	// 	Data             [][]float64
}

// Open opens mbtiles at given path
func Open(mbTilesPath string, name string, format string) (MBTiles, error) {
	var mbTiles MBTiles

	db, err := sql.Open("sqlite3", mbTilesPath)
	if err != nil {
		fmt.Println("0")
		return mbTiles, err
	}

	_, err = db.Exec(`
		PRAGMA application_id = 0x4d504258;
		CREATE TABLE IF NOT EXISTS metadata (name text, value text);
		CREATE TABLE IF NOT EXISTS tiles (zoom_level integer, tile_column integer, tile_row integer, tile_data blob);
		CREATE UNIQUE INDEX IF NOT EXISTS tile_index on tiles (zoom_level, tile_column, tile_row);
	`)
	if err != nil {
		db.Close()
		fmt.Println("1")
		return mbTiles, err
	}

	tileInsertStmt, err := db.Prepare("INSERT OR REPLACE INTO tiles (zoom_level, tile_column, tile_row, tile_data) VALUES (?, ?, ?, ?);")
	if err != nil {
		db.Close()
		fmt.Println("2")
		return mbTiles, err
	}

	mbTiles.db = db
	mbTiles.tileInsertStmt = tileInsertStmt

	// set name format (and - if required - json)
	metas := [][2]string{
		[2]string{"name", name},
		[2]string{"format", format},
	}
	if format == "pbf" {
		metas = append(metas, [2]string{"json", `{ "vector_layers": [] }`})
	}
	err = mbTiles.InsertMeta(metas)
	if err != nil {
		db.Close()
		fmt.Println("3")
		return mbTiles, err
	}

	return mbTiles, nil
}

// Close releases db file
func (mbTiles MBTiles) Close() error {
	err := mbTiles.tileInsertStmt.Close()
	if err != nil {
		return err
	}

	return mbTiles.db.Close()
}

// InsertTile inserts a tile at (z, x, y).
func (mbTiles MBTiles) InsertTile(z, x, y uint, tileData []byte) error {
	_, err := mbTiles.tileInsertStmt.Exec(z, x, y, tileData)
	return err
}

// InsertMeta set metadata entry
func (mbTiles MBTiles) InsertMeta(entries [][2]string) error {
	values := make([]string, len(entries))

	for i, entry := range entries {
		n := entry[0]
		v := entry[1]

		values[i] = fmt.Sprintf("('%s', '%s')", n, v)
	}

	_, err := mbTiles.db.Exec(fmt.Sprintf("INSERT OR REPLACE INTO metadata (name, value) VALUES %s ;", strings.Join(values, ", ")))
	return err
}
