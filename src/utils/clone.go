package utils

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
)

// CloneFeature deep clones given feature
func CloneFeature(f *geojson.Feature) *geojson.Feature {
	newFeature := geojson.NewFeature(orb.Clone(f.Geometry))

	newFeature.ID = f.ID
	newFeature.Type = f.Type
	newFeature.Properties = f.Properties.Clone()
	copy(newFeature.BBox, f.BBox)

	return newFeature
}

// DeepCloneFeatureCollection deep clones given feature collection
func DeepCloneFeatureCollection(fc *geojson.FeatureCollection) *geojson.FeatureCollection {

	features := make([]*geojson.Feature, len(fc.Features))

	for i := 0; i < len(fc.Features); i++ {
		features[i] = CloneFeature(fc.Features[i])
	}

	newFc := geojson.NewFeatureCollection()

	newFc.Features = features
	copy(newFc.BBox, fc.BBox)
	newFc.Type = fc.Type

	return newFc
}

// DeepCloneLayers deep clones given layers
func DeepCloneLayers(l mvt.Layers) mvt.Layers {

	fcs := l.ToFeatureCollections()

	for key, fc := range fcs {
		fcs[key] = DeepCloneFeatureCollection(fc)
	}

	newL := mvt.NewLayers(fcs)

	return newL
}
