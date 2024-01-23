package geoutil

import (
	"fmt"
	"math"
	"time"

	"github.com/andrepxx/location-visualizer/geo"
	"github.com/andrepxx/location-visualizer/geo/geodb"
)

const (
	BLOCK_SIZE                  = 1024
	DEGREES_TO_RADIANS          = math.Pi / 180.0
	DEGREES_E7_TO_RADIANS       = DEGREES_TO_RADIANS * 1e-7
	IMPORT_ALL                  = 1
	IMPORT_NEWER                = 2
	IMPORT_NONE                 = 0
	MILLISECONDS_PER_SECOND     = 1000
	NANOSECONDS_PER_MILLISECOND = 1000000
)

/*
 * Statistics for a geographical dataset.
 */
type DatasetStats interface {
	LocationCount() uint32
	Ordered() bool
	OrderedStrict() bool
	TimestampEarliest() uint64
	TimestampLatest() uint64
}

/*
 * A report for a data migration.
 */
type MigrationReport interface {
	After() DatasetStats
	Before() DatasetStats
	Imported() DatasetStats
	Source() DatasetStats
}

/*
 * A utility for transforming geographic data.
 */
type Util interface {
	DegreesE7ToRadians(degreesE7 int32) float64
	GeoDBStats(db geodb.Database) (DatasetStats, error)
	GeoJSONOrGPXStats(db geo.Database) (DatasetStats, error)
	Migrate(dst geodb.Database, src geo.Database, importStrategy int) (MigrationReport, error)
	MillisecondsToTime(ms uint64) time.Time
}

/*
 * Data structure representing statistics for a geographical dataset.
 */
type datasetStatsStruct struct {
	locationCount     uint32
	ordered           bool
	orderedStrict     bool
	timestampEarliest uint64
	timestampLatest   uint64
}

/*
 * Data structure representing a migration report.
 */
type migrationReportStruct struct {
	after    datasetStatsStruct
	before   datasetStatsStruct
	imported datasetStatsStruct
	source   datasetStatsStruct
}

/*
 * Data structure representing a geo utility.
 */
type utilStruct struct {
}

/*
 * Returns the number of locations in the data set.
 */
func (this *datasetStatsStruct) LocationCount() uint32 {
	locationCount := this.locationCount
	return locationCount
}

/*
 * Returns whether the data set is ordered by time stamp, that is, the
 * following implication is true for all elements in the data set:
 *
 * (i > j) --> (data[i].timestamp >= data[j].timestamp)
 */
func (this *datasetStatsStruct) Ordered() bool {
	ordered := this.ordered
	return ordered
}

/*
 * Returns whether the data set is strictly ordered by time stamp, that is, the
 * following implication is true for all elements in the data set:
 *
 * (i > j) --> (data[i].timestamp > data[j].timestamp)
 */
func (this *datasetStatsStruct) OrderedStrict() bool {
	orderedStrict := this.orderedStrict
	return orderedStrict
}

/*
 * Returns the earliest (smallest) timestamp in the data set.
 */
func (this *datasetStatsStruct) TimestampEarliest() uint64 {
	timestampEarliest := this.timestampEarliest
	return timestampEarliest
}

/*
 * Returns the latest (largest) timestamp in the data set.
 */
func (this *datasetStatsStruct) TimestampLatest() uint64 {
	timestampLatest := this.timestampLatest
	return timestampLatest
}

/*
 * Returns statistics about the state of the target data set after migration
 * was finished.
 */
func (this *migrationReportStruct) After() DatasetStats {
	after := &this.after
	return after
}

/*
 * Returns statistics about the state of the target data set before migration
 * was started.
 */
func (this *migrationReportStruct) Before() DatasetStats {
	before := &this.before
	return before
}

/*
 * Returns statistics about the records which got migrated from the source data
 * set into the target data set.
 */
func (this *migrationReportStruct) Imported() DatasetStats {
	imported := &this.imported
	return imported
}

/*
 * Returns statistics about the records provided in the source data set.
 */
func (this *migrationReportStruct) Source() DatasetStats {
	source := &this.source
	return source
}

/*
 * Internal function to create statistics from a GeoDB database.
 *
 * The contents of the database may not change while this function runs, i. e.
 * the database must be locked for reading.
 */
func (this *utilStruct) geoDBStats(db geodb.Database) (datasetStatsStruct, error) {

	/*
	 * Query database if it is non-nil.
	 */
	if db == nil {
		return datasetStatsStruct{}, fmt.Errorf("%s", "Database is nil!")
	} else {
		locationCount := db.LocationCount()
		ordered := true
		orderedStrict := true
		timestampEarliest := uint64(math.MaxUint64)
		timestampLatest := uint64(0)
		timestampOld := uint64(0)
		locations := make([]geodb.Location, BLOCK_SIZE)
		idx := uint32(0)
		errDatabase := error(nil)

		/*
		 * Read until end or database error occurs.
		 */
		for (idx < locationCount) && (errDatabase == nil) {
			n, err := db.ReadLocations(idx, locations)

			/*
			 * Iterate over the locations.
			 */
			for i := uint32(0); i < n; i++ {
				location := &locations[i]
				timestamp := location.Timestamp

				/*
				 * Check if we found an earlier timestamp.
				 */
				if timestamp < timestampEarliest {
					timestampEarliest = timestamp
				}

				/*
				 * Check if we found a later timestamp.
				 */
				if timestamp > timestampLatest {
					timestampLatest = timestamp
				}

				ordered = ordered && (timestamp >= timestampOld)
				orderedStrict = orderedStrict && (timestamp > timestampOld)
				timestampOld = timestamp
			}

			idx += n
			errDatabase = err
		}

		/*
		 * Check if database error occured.
		 */
		if errDatabase != nil {
			msg := errDatabase.Error()
			return datasetStatsStruct{}, fmt.Errorf("Error accessing database: %s", msg)
		} else {

			/*
			 * Create data structure for statistics.
			 */
			stats := datasetStatsStruct{
				locationCount:     locationCount,
				ordered:           ordered,
				orderedStrict:     orderedStrict,
				timestampEarliest: timestampEarliest,
				timestampLatest:   timestampLatest,
			}

			return stats, nil
		}

	}

}

/*
 * Internal function to create statistics from a GeoJSON database.
 */
func (this *utilStruct) geoJSONOrGPXStats(db geo.Database) (datasetStatsStruct, error) {

	/*
	 * Query database if it is non-nil.
	 */
	if db == nil {
		return datasetStatsStruct{}, fmt.Errorf("%s", "Database is nil!")
	} else {
		locationCount := db.LocationCount()
		ordered := true
		orderedStrict := true
		timestampEarliest := uint64(math.MaxUint64)
		timestampLatest := uint64(0)
		timestampOld := uint64(0)
		errDatabase := error(nil)

		/*
		 * Iterate over the locations.
		 */
		for idx := 0; (idx < locationCount) && (errDatabase == nil); idx++ {
			location, err := db.LocationAt(idx)

			/*
			 * Check if database error occured.
			 */
			if err != nil {
				errDatabase = err
			} else {
				timestamp := location.Timestamp()

				/*
				 * Check if we found an earlier timestamp.
				 */
				if timestamp < timestampEarliest {
					timestampEarliest = timestamp
				}

				/*
				 * Check if we found a later timestamp.
				 */
				if timestamp > timestampLatest {
					timestampLatest = timestamp
				}

				ordered = ordered && (timestamp >= timestampOld)
				orderedStrict = orderedStrict && (timestamp > timestampOld)
				timestampOld = timestamp
			}

		}

		/*
		 * Check if database error occured.
		 */
		if errDatabase != nil {
			msg := errDatabase.Error()
			return datasetStatsStruct{}, fmt.Errorf("Error accessing database: %s", msg)
		} else {
			locationCount32 := uint32(locationCount)

			/*
			 * Check for overflow.
			 */
			if locationCount > math.MaxUint32 {
				locationCount32 = math.MaxUint32
			}

			/*
			 * Create data structure for statistics.
			 */
			stats := datasetStatsStruct{
				locationCount:     locationCount32,
				ordered:           ordered,
				orderedStrict:     orderedStrict,
				timestampEarliest: timestampEarliest,
				timestampLatest:   timestampLatest,
			}

			return stats, nil
		}

	}

}

/*
 * Convert an angle from degrees in fixed-point representation with a fixed
 * exponent of seven to radians in floating-point representation.
 */
func (this *utilStruct) DegreesE7ToRadians(degreesE7 int32) float64 {
	degreesE7Float := float64(degreesE7)
	result := DEGREES_E7_TO_RADIANS * degreesE7Float
	return result
}

/*
 * Create statistics from a GeoDB database.
 *
 * The contents of the GeoDB database may not change while this function runs,
 * i. e. the GeoDB database must be locked for reading.
 */
func (this *utilStruct) GeoDBStats(db geodb.Database) (DatasetStats, error) {
	stats, err := this.geoDBStats(db)

	/*
	 * Return nil stats if error occured.
	 */
	if err != nil {
		return nil, err
	} else {
		return &stats, nil
	}

}

/*
 * Create statistics from a GeoJSON or GPX database.
 */
func (this *utilStruct) GeoJSONOrGPXStats(db geo.Database) (DatasetStats, error) {
	stats, err := this.geoJSONOrGPXStats(db)

	/*
	 * Return nil stats if error occured.
	 */
	if err != nil {
		return nil, err
	} else {
		return &stats, nil
	}

}

/*
 * Migrate data from a GeoJSON / GPX database to a GeoDB database.
 */
func (this *utilStruct) Migrate(dst geodb.Database, src geo.Database, importStrategy int) (MigrationReport, error) {
	errResult := error(nil)
	statsImported := datasetStatsStruct{}
	statsBefore, errBefore := this.geoDBStats(dst)
	statsSource, errSource := this.geoJSONOrGPXStats(src)

	/*
	 * Check if GeoDB and GeoJSON databases could be accessed.
	 */
	if errBefore != nil {
		msg := errBefore.Error()
		errResult = fmt.Errorf("Error accessing GeoDB database: %s", msg)
	} else if errSource != nil {
		msg := errSource.Error()
		errResult = fmt.Errorf("Error accessing GeoJSON database: %s", msg)
	} else {
		locationCount := uint64(0)
		ordered := true
		orderedStrict := true
		timestampEarliest := uint64(math.MaxUint64)
		timestampLatest := uint64(0)
		timestampOld := uint64(0)
		errDatabaseSource := error(nil)
		errDatabaseTarget := error(nil)
		locationCountSource := src.LocationCount()
		timestampLatestBeforeImport := statsBefore.TimestampLatest()

		/*
		 * Import locations from GeoJSON database.
		 */
		for i := 0; i < locationCountSource; i++ {
			locationSource, errRead := src.LocationAt(i)

			/*
			 * Check for read errors.
			 */
			if errRead != nil {
				errDatabaseSource = errRead
			} else {
				timestamp := locationSource.Timestamp()
				migrate := false

				/*
				 * Decide on the chosen import strategy.
				 */
				switch importStrategy {
				case IMPORT_ALL:
					migrate = true
				case IMPORT_NEWER:
					migrate = timestamp > timestampLatestBeforeImport
				default:
					// Do nothing.
				}

				/*
				 * Check if we shall migrate this record.
				 */
				if migrate {

					/*
					 * Check if we found an earlier timestamp.
					 */
					if timestamp < timestampEarliest {
						timestampEarliest = timestamp
					}

					/*
					 * Check if we found a later timestamp.
					 */
					if timestamp > timestampLatest {
						timestampLatest = timestamp
					}

					ordered = ordered && (timestamp >= timestampOld)
					orderedStrict = orderedStrict && (timestamp > timestampOld)
					timestampOld = timestamp
					latitude := locationSource.Latitude()
					longitude := locationSource.Longitude()

					/*
					 * Create GeoDB location.
					 */
					locationTarget := geodb.Location{
						Timestamp:   timestamp,
						LatitudeE7:  latitude,
						LongitudeE7: longitude,
					}

					errWrite := dst.Append(&locationTarget)

					/*
					 * Check for write errors.
					 */
					if errWrite != nil {
						errDatabaseTarget = errWrite
					} else {
						locationCount++
					}

				}

			}

		}

		/*
		 * Check for database error.
		 */
		if errDatabaseSource != nil {
			msg := errDatabaseSource.Error()
			errResult = fmt.Errorf("Error reading from GeoJSON database: %s", msg)
		} else if errDatabaseTarget != nil {
			msg := errDatabaseTarget.Error()
			errResult = fmt.Errorf("Error writing to GeoDB database: %s", msg)
		}

		locationCount32 := uint32(locationCount)

		/*
		 * Check for overflow.
		 */
		if locationCount > math.MaxUint32 {
			locationCount32 = math.MaxUint32
		}

		/*
		 * Create statistics about imported data sets.
		 */
		statsImported = datasetStatsStruct{
			locationCount:     locationCount32,
			ordered:           ordered,
			orderedStrict:     orderedStrict,
			timestampEarliest: timestampEarliest,
			timestampLatest:   timestampLatest,
		}

	}

	statsAfter, errAfter := this.geoDBStats(dst)

	/*
	 * Check for database error.
	 */
	if (errAfter != nil) && (errResult == nil) {
		msg := errAfter.Error()
		errResult = fmt.Errorf("Error accessing GeoDB database: %s", msg)
	}

	/*
	 * Create data migration report.
	 */
	migrationReport := migrationReportStruct{
		after:    statsAfter,
		before:   statsBefore,
		imported: statsImported,
		source:   statsSource,
	}

	return &migrationReport, errResult
}

/*
 * Convert a timestamp from milliseconds since the Epoch into a time.Time.
 */
func (this *utilStruct) MillisecondsToTime(ms uint64) time.Time {
	s := int64(ms / MILLISECONDS_PER_SECOND)
	ns := int64((ms % MILLISECONDS_PER_SECOND) * NANOSECONDS_PER_MILLISECOND)
	t := time.Unix(s, ns)
	utc := t.UTC()
	return utc
}

/*
 * Creates a utility for working with geographic databases.
 */
func Create() Util {
	util := utilStruct{}
	return &util
}
