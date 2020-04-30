package coords

import (
	"errors"
	"fmt"
	"math"
)

/**
 *      We use slightly modified version of the formulas to calculate the pixel coordinates with a specific zoom level of latitude/longitude
 *
 *      Original Formulas (https://en.m.wikipedia.org/wiki/Web_Mercator_projection#Formulas):
 *          x = (256 / 2π) * 2^z * (longitude + π)
 *          y = (256 / 2π) * 2^z * (π - ln[ tan(π/4 + latitude/2) ])
 *      with
 *          - z = zoom level
 *          - lat/lon are in radians
 *          - (0,0) is in the upper left corner
 *          - (256,256) in the lower right corner
 *
 *      Modifications:
 *          - We'll use zoom level 0, because it covers the whole coordinate system
 *          - instead of using 256 as the max size we'll use the worldSize
 *          - instead of the origin (0,0) being in the upper left corner, we want it to be - like in Arma's coordinate system - in the bottom left corner
 *
 *      So we end up with the following formulas to convert from web mercator latitude/longitude to Arma's coordinate space:
 *          x = (worldSize / 2π) * (longitude + π)
 *          y = worldSize - (worldSize / 2π) * (π - ln[ tan(π/4 + latitude/2) ])
 *
 *      To convert from Arma's coordinate space to web mercator you can just solve the formulas for latitude/longitude and end up with the following:
 *          longitude = π * (2x / worldSize - 1)
 *          latitude = 2 * atan(e^(2πy/w - π)) - π/2
 *
 */

var latMax = float64(2)*math.Atan(math.Pow(math.E, math.Pi)) - math.Pi/2

func rad2deg(rad float64) float64 { return (rad * (180.0 / math.Pi)) }
func deg2rad(deg float64) float64 { return (deg * (math.Pi / 180.0)) }

// LatLng holds latitude and longitude
type LatLng struct {
	Latitude  float64
	Longitude float64
}

// ArmaXY represents a position on an Arma map
type ArmaXY struct {
	X float64
	Y float64
}

// LatLng2Arma converts web mercator (EPSG:3857) latitude longitude to Arma coordinates
func LatLng2Arma(worldSize float64, latLng LatLng) (ArmaXY, error) {

	if worldSize <= 0 {
		return ArmaXY{0, 0}, errors.New("worldSize must be larger than 0")
	}

	if latLng.Latitude > latMax {
		return ArmaXY{0, 0}, fmt.Errorf("latitude must not be larger than %f", latMax)
	}

	var x, y float64

	x = worldSize / (2 * math.Pi) * (deg2rad(latLng.Longitude) + math.Pi)
	y = worldSize - (worldSize/(2*math.Pi))*(math.Pi-math.Log(math.Tan(math.Pi/4+deg2rad(latLng.Latitude)/2)))

	return ArmaXY{x, y}, nil
}

// Arma2LatLng converts arma coordinates to web mercator (EPSG:3857) latitude longitude
func Arma2LatLng(worldSize float64, pos ArmaXY) (LatLng, error) {

	if worldSize <= 0 {
		return LatLng{0, 0}, errors.New("worldSize must be larger than 0")
	}

	var latitude, longitude float64

	latitude = 2*math.Atan(math.Pow(math.E, math.Pi*((2*pos.Y)/worldSize-1))) - math.Pi/2
	longitude = math.Pi * ((2*pos.X)/worldSize - 1)

	return LatLng{rad2deg(latitude), rad2deg(longitude)}, nil
}
