package mvt

import (
	"../coords"
	"errors"
	orb "github.com/paulmach/orb"
	geojson "github.com/paulmach/orb/geojson"
	"log"
)

func convertLayer(fc *geojson.FeatureCollection, worldSize float64) {
	for i := 0; i < len((*fc).Features); i++ {
		feature := &((*fc).Features)[i]
		(*feature).Geometry = convertGeoCRS(&(*feature).Geometry, worldSize)
	}
}

func convertGeoCRS(geoPtr *orb.Geometry, worldSize float64) orb.Geometry {
	convertPoint := func(worldSize float64, point orb.Point) orb.Point {
		latLng, err := coords.Arma2LatLng(worldSize, coords.ArmaXY{X: point.X(), Y: point.Y()})

		if err != nil {
			log.Fatal(errors.New("Failed to convert coordinates"))
		}

		return orb.Point{latLng.Longitude, latLng.Latitude}
	}

	switch (*geoPtr).GeoJSONType() {
	case "Point":
		point := (*geoPtr).(orb.Point)
		return convertPoint(worldSize, point)
	case "LineString":
		lineString := (*geoPtr).(orb.LineString)
		for i := 0; i < len(lineString); i++ {
			lineString[i] = convertPoint(worldSize, lineString[i])
		}
		return lineString
	case "Polygon":
		polygon := (*geoPtr).(orb.Polygon)
		for i := 0; i < len(polygon); i++ {
			for j := 0; j < len(polygon[i]); j++ {
				polygon[i][j] = convertPoint(worldSize, polygon[i][j])
			}
		}
		return polygon
	case "MultiPoint":
		multiPoint := (*geoPtr).(orb.MultiPoint)
		for i := 0; i < len(multiPoint); i++ {
			multiPoint[i] = convertPoint(worldSize, multiPoint[i])
		}
		return multiPoint
	case "MultiLineString":
		multiLineString := (*geoPtr).(orb.MultiLineString)
		for i := 0; i < len(multiLineString); i++ {
			for j := 0; j < len(multiLineString[i]); j++ {
				multiLineString[i][j] = convertPoint(worldSize, multiLineString[i][j])
			}
		}
		return multiLineString
	case "MultiPolygon":
		multiPolygon := (*geoPtr).(orb.MultiPolygon)
		for i := 0; i < len(multiPolygon); i++ {
			for j := 0; j < len(multiPolygon[i]); j++ {
				for k := 0; k < len(multiPolygon[i][j]); k++ {
					multiPolygon[i][j][k] = convertPoint(worldSize, multiPolygon[i][j][k])
				}
			}
		}
		return multiPolygon
	}

	return nil
}
