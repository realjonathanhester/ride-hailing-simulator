// Package polyline encodes points in the Google encoded polyline format (precision 5).
package polyline

import (
	"math"
	"strings"

	"github.com/realjonathanhester/ride-hailing-simulator/model"
)

// Encode returns the encoded polyline string for points.
func Encode(points []model.Location) string {
	var b strings.Builder
	var lastLat, lastLng int64
	for _, p := range points {
		lat := int64(math.Round(p.Lat * 1e5))
		lng := int64(math.Round(p.Lng * 1e5))
		encodeValue(&b, lat-lastLat)
		encodeValue(&b, lng-lastLng)
		lastLat, lastLng = lat, lng
	}
	return b.String()
}

func encodeValue(b *strings.Builder, v int64) {
	v <<= 1
	if v < 0 {
		v = ^v
	}
	for v >= 0x20 {
		b.WriteByte(byte((0x20 | (v & 0x1f)) + 63))
		v >>= 5
	}
	b.WriteByte(byte(v + 63))
}

// Decode parses an encoded polyline string (precision 5).
func Decode(encoded string) []model.Location {
	var points []model.Location
	var lat, lng int64
	for i := 0; i < len(encoded); {
		var deltas [2]int64
		for k := 0; k < 2; k++ {
			var result int64
			shift := uint(0)
			for i < len(encoded) {
				b := int64(encoded[i]) - 63
				i++
				result |= (b & 0x1f) << shift
				shift += 5
				if b < 0x20 {
					break
				}
			}
			if result&1 != 0 {
				deltas[k] = ^(result >> 1)
			} else {
				deltas[k] = result >> 1
			}
		}
		lat += deltas[0]
		lng += deltas[1]
		points = append(points, model.Location{Lat: float64(lat) / 1e5, Lng: float64(lng) / 1e5})
	}
	return points
}

// earthRadiusMeters is the mean Earth radius used by HaversineMeters.
const earthRadiusMeters = 6371008.8

// HaversineMeters is the great-circle distance between a and b in meters.
func HaversineMeters(a, b model.Location) float64 {
	const radians = math.Pi / 180
	dLat := (b.Lat - a.Lat) * radians
	dLng := (b.Lng - a.Lng) * radians
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(a.Lat*radians)*math.Cos(b.Lat*radians)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(math.Min(1, h)))
}

// Thin drops points that are less than minMeters (haversine) from the last
// kept point. The first and last points are always kept, and so is every
// point for which keep(i) is true (keep may be nil). When a point that must
// be kept lands within minMeters of the previous kept point, that previous
// point is dropped instead unless it must be kept too, so consecutive points
// are at least minMeters apart except between two points that must be kept.
func Thin(points []model.Location, minMeters float64, keep func(i int) bool) []model.Location {
	if len(points) <= 2 {
		return points
	}
	thinned := []model.Location{points[0]}
	pinned := []bool{true}
	for i := 1; i < len(points); i++ {
		required := i == len(points)-1 || (keep != nil && keep(i))
		last := len(thinned) - 1
		if HaversineMeters(thinned[last], points[i]) >= minMeters {
			thinned, pinned = append(thinned, points[i]), append(pinned, required)
			continue
		}
		if !required {
			continue
		}
		for last > 0 && !pinned[last] && HaversineMeters(thinned[last], points[i]) < minMeters {
			thinned, pinned = thinned[:last], pinned[:last]
			last--
		}
		thinned, pinned = append(thinned, points[i]), append(pinned, true)
	}
	return thinned
}
