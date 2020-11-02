package mvt

import (
	"encoding/json"
	"log"
	"os"
	"path"
)

type vectorLayer struct {
	ID     string            `json:"id"`
	Fields map[string]string `json:"fields"`
}

type tileJSON struct {
	Minzoom      uint16        `json:"minzoom"`
	Maxzoom      uint16        `json:"maxzoom"`
	VectorLayers []vectorLayer `json:"vector_layers"`
}

var contourFields = map[string]string{"elevation": "Elevation of contour."}

// TODO:
var locationFields = map[string]string{
	"name":    "Corresponds to value in map config.",
	"radiusA": "Corresponds to value in map config.",
	"radiusB": "Corresponds to value in map config.",
	"angle":   "Corresponds to value in map config.",
}

var fields = map[string]map[string]string{
	"contours/01":                   contourFields,
	"contours/05":                   contourFields,
	"contours/10":                   contourFields,
	"contours/100":                  contourFields,
	"contours/50":                   contourFields,
	"house":                         {"color": "House color as a CSS rgb() string."},
	"locations/respawn_unknown":     locationFields,
	"locations/respawn_inf":         locationFields,
	"locations/respawn_motor":       locationFields,
	"locations/respawn_armor":       locationFields,
	"locations/respawn_air":         locationFields,
	"locations/respawn_plane":       locationFields,
	"locations/respawn_naval":       locationFields,
	"locations/respawn_para":        locationFields,
	"locations/group_0":             locationFields,
	"locations/group_1":             locationFields,
	"locations/group_2":             locationFields,
	"locations/group_3":             locationFields,
	"locations/group_4":             locationFields,
	"locations/group_5":             locationFields,
	"locations/group_6":             locationFields,
	"locations/group_7":             locationFields,
	"locations/group_8":             locationFields,
	"locations/group_9":             locationFields,
	"locations/group_10":            locationFields,
	"locations/group_11":            locationFields,
	"locations/b_unknown":           locationFields,
	"locations/o_unknown":           locationFields,
	"locations/n_unknown":           locationFields,
	"locations/b_inf":               locationFields,
	"locations/o_inf":               locationFields,
	"locations/n_inf":               locationFields,
	"locations/b_motor_inf":         locationFields,
	"locations/o_motor_inf":         locationFields,
	"locations/n_motor_inf":         locationFields,
	"locations/b_mech_inf":          locationFields,
	"locations/o_mech_inf":          locationFields,
	"locations/n_mech_inf":          locationFields,
	"locations/b_armor":             locationFields,
	"locations/o_armor":             locationFields,
	"locations/n_armor":             locationFields,
	"locations/b_recon":             locationFields,
	"locations/o_recon":             locationFields,
	"locations/n_recon":             locationFields,
	"locations/b_air":               locationFields,
	"locations/o_air":               locationFields,
	"locations/n_air":               locationFields,
	"locations/b_plane":             locationFields,
	"locations/o_plane":             locationFields,
	"locations/n_plane":             locationFields,
	"locations/b_uav":               locationFields,
	"locations/o_uav":               locationFields,
	"locations/n_uav":               locationFields,
	"locations/b_naval":             locationFields,
	"locations/o_naval":             locationFields,
	"locations/n_naval":             locationFields,
	"locations/b_med":               locationFields,
	"locations/o_med":               locationFields,
	"locations/n_med":               locationFields,
	"locations/b_art":               locationFields,
	"locations/o_art":               locationFields,
	"locations/n_art":               locationFields,
	"locations/b_mortar":            locationFields,
	"locations/o_mortar":            locationFields,
	"locations/n_mortar":            locationFields,
	"locations/b_hq":                locationFields,
	"locations/o_hq":                locationFields,
	"locations/n_hq":                locationFields,
	"locations/b_support":           locationFields,
	"locations/o_support":           locationFields,
	"locations/n_support":           locationFields,
	"locations/b_maint":             locationFields,
	"locations/o_maint":             locationFields,
	"locations/n_maint":             locationFields,
	"locations/b_service":           locationFields,
	"locations/o_service":           locationFields,
	"locations/n_service":           locationFields,
	"locations/b_installation":      locationFields,
	"locations/o_installation":      locationFields,
	"locations/n_installation":      locationFields,
	"locations/u_installation":      locationFields,
	"locations/b_antiair":           locationFields,
	"locations/o_antiair":           locationFields,
	"locations/n_antiair":           locationFields,
	"locations/c_unknown":           locationFields,
	"locations/c_car":               locationFields,
	"locations/c_ship":              locationFields,
	"locations/c_air":               locationFields,
	"locations/c_plane":             locationFields,
	"locations/flag":                locationFields,
	"locations/rockarea":            locationFields,
	"locations/viewpoint":           locationFields,
	"locations/hill":                locationFields,
	"locations/bordercrossing":      locationFields,
	"locations/vegetationbroadleaf": locationFields,
	"locations/vegetationfir":       locationFields,
	"locations/vegetationpalm":      locationFields,
	"locations/vegetationvineyard":  locationFields,
	"locations/handdrawncamp":       locationFields,
	"locations/name":                locationFields,
	"locations/faketown":            locationFields,
	"locations/strategic":           locationFields,
	"locations/strongpointarea":     locationFields,
	"locations/flatarea":            locationFields,
	"locations/flatareacity":        locationFields,
	"locations/flatareacitysmall":   locationFields,
	"locations/citycenter":          locationFields,
	"locations/civildefense":        locationFields,
	"locations/culturalproperty":    locationFields,
	"locations/dangerousforces":     locationFields,
	"locations/safetyzone":          locationFields,
	"locations/airport":             locationFields,
	"locations/namemarine":          locationFields,
	"locations/namelocal":           locationFields,
	"locations/namevillage":         locationFields,
	"locations/namecity":            locationFields,
	"locations/namecitycapital":     locationFields,
	"locations/mount":               locationFields,
	"locations/invisible":           locationFields,
	"locations/historicalsite":      locationFields,
	"locations/area":                locationFields,
}

// writeTileJSON writes sat.json containing the maxLod to the sat.json into the outputDirectory
func writeTileJSON(outputDirectory string, maxLod uint16, layerNames []string) {
	var err error

	vectorLayers := make([]vectorLayer, len(layerNames))

	for i, layerName := range layerNames {
		vectorLayers[i] = nameToLayer(layerName)
	}

	obj := tileJSON{
		Minzoom:      0,
		Maxzoom:      maxLod,
		VectorLayers: vectorLayers,
	}

	// create file
	f, err := os.Create(path.Join(outputDirectory, "tile.json"))
	if err != nil {
		log.Fatal(err)
	}

	// marshal
	bytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	// write file
	_, err = f.Write(bytes)
	if err != nil {
		log.Fatal(err)
	}

	// close file
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func nameToLayer(layerName string) vectorLayer {
	fields, found := fields[layerName]

	if !found {
		fields = map[string]string{}
	}

	return vectorLayer{
		ID:     layerName,
		Fields: fields,
	}
}
