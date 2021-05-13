package tilejson

// VectorLayer represents a vector layer of a tile.json
type VectorLayer struct {
	ID     string            `json:"id"`
	Fields map[string]string `json:"fields"`
}

// TileJSON represents a tile.json
type TileJSON struct {
	TileJSON     string        `json:"tilejson"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Scheme       string        `json:"scheme"`
	Minzoom      uint8         `json:"minzoom"`
	Maxzoom      uint8         `json:"maxzoom"`
	VectorLayers []VectorLayer `json:"vector_layers,omitempty"`
}
