package meta

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/andrepxx/location-visualizer/filter"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*
 * Global constants.
 */
const (
	EXPECTED_NUM_FIELDS = 10
	LOWER_BEFORE_SHIFT  = (math.MaxUint64 / 10) + 1
	REX_FLOAT           = "^(\\s*)(\\d*)(\\.?)(\\d*)(\\s*)$"
	TIME_DAY            = 24 * time.Hour
)

/*
 * The activity of running.
 */
type RunningActivity interface {
	DistanceKM() string
	Duration() time.Duration
	EnergyKJ() uint64
	StepCount() uint64
	Zero() bool
}

/*
 * The activity of cycling.
 */
type CyclingActivity interface {
	DistanceKM() string
	Duration() time.Duration
	EnergyKJ() uint64
	Zero() bool
}

/*
 * Activities other than running and cycling.
 */
type OtherActivity interface {
	EnergyKJ() uint64
	Zero() bool
}

/*
 * An activity group is a set of activities carried out within a specific time
 * interval, typically a day.
 */
type ActivityGroup interface {
	Begin() time.Time
	Cycling() CyclingActivity
	Other() OtherActivity
	Running() RunningActivity
	WeightKG() string
}

/*
 * Data structure to obtain information about activities from external caller.
 *
 * This is used to reduce the number of parameters passed to the method
 * Activities.Add(...).
 */
type ActivityInfo struct {
	Begin             time.Time
	WeightKG          string
	RunningDuration   time.Duration
	RunningDistanceKM string
	RunningStepCount  uint64
	RunningEnergyKJ   uint64
	CyclingDuration   time.Duration
	CyclingDistanceKM string
	CyclingEnergyKJ   uint64
	OtherEnergyKJ     uint64
}

/*
 * All activities about which information can be stored.
 */
type Activities interface {
	Add(info *ActivityInfo) error
	End(id uint32) (time.Time, error)
	Export() ([]byte, error)
	Get(id uint32) (ActivityGroup, error)
	Import(buf []byte) error
	ImportCSV(data string) error
	Length() uint32
	Remove(id uint32) error
	Replace(id uint32, info *ActivityInfo) error
	Revision() uint64
}

/*
 * An unsigned fixed-point number.
 */
type unsignedFixed struct {
	mantissa uint64
	exponent uint8
}

/*
 * Data structure storing information about a running activity.
 */
type runningActivityStruct struct {
	duration   time.Duration
	distanceKM unsignedFixed
	stepCount  uint64
	energyKJ   uint64
}

/*
 * Data structure storing information about a cycling activity.
 */
type cyclingActivityStruct struct {
	duration   time.Duration
	distanceKM unsignedFixed
	energyKJ   uint64
}

/*
 * Data structure representing activities not captured by more specific
 * activity structs (i. e. currently all others than running and cycling).
 *
 * It mainly accounts for the amount of energy consumed by the human body while
 * not carrying out one of the more specific activities, in order to arrive at
 * a plausible total amount of energy consumption during a certain period of
 * activity (e. g. a day).
 */
type otherActivityStruct struct {
	energyKJ uint64
}

/*
 * Data structure representing an activity group.
 *
 * An activity group is a set of activities carried out within a specific time
 * interval, typically a day.
 */
type activityGroupStruct struct {
	begin    time.Time
	weightKG unsignedFixed
	running  runningActivityStruct
	cycling  cyclingActivityStruct
	other    otherActivityStruct
}

/*
 * Data structure storing all activities.
 */
type activitiesStruct struct {
	mutex    sync.RWMutex
	groups   []activityGroupStruct
	revision uint64
}

/*
 * Parse an unsigned fixed-point number with a given number of decimal places
 * from a string representation.
 */
func parseUnsignedFixed(value string, decimalPlaces uint8) (unsignedFixed, error) {
	val := strings.TrimSpace(value)
	rex, _ := regexp.Compile(REX_FLOAT)

	/*
	 * Check if regular expression compiles.
	 */
	if rex == nil {
		return unsignedFixed{}, fmt.Errorf("Failed to compile regular expression: '%s'", REX_FLOAT)
	} else {
		matches := rex.MatchString(val)

		/*
		 * Check syntax of number.
		 */
		if !matches {
			return unsignedFixed{}, fmt.Errorf("Value '%s' does not match regular expression '%s'.", val, REX_FLOAT)
		} else {
			vi := uint64(0)
			exp := uint8(0)
			dot := false
			fail := false

			/*
			 * Iterate over the bytes in the string.
			 */
			for _, c := range []byte(value) {

				/*
				 * Do this as long as we're not in failure mode.
				 */
				if !fail {

					/*
					 * Check if we have to read more digits.
					 */
					if !dot || (exp < decimalPlaces) {

						/*
						 * Handle decimal digit.
						 */
						if ('0' <= c) && (c <= '9') {

							/*
							 * Handle overflow before multiplication.
							 */
							if vi >= LOWER_BEFORE_SHIFT {
								vi = math.MaxUint64
								fail = true
							} else {
								vi *= 10

								/*
								 * If we already read decimal dot, increment exponent.
								 */
								if dot {
									exp++
								}

							}

							digit := c - '0'
							digit64 := uint64(digit)
							vip := vi + digit64

							/*
							 * Handle overflow before addition.
							 */
							if vip < vi {
								vi = math.MaxUint64
								fail = true
							} else {
								vi = vip
							}

						}

						/*
						 * Handle dot.
						 */
						if c == '.' {
							dot = true
						}

					}

				}

			}

			/*
			 * Shift the number to the right amount of decimal places.
			 */
			for exp < decimalPlaces {

				/*
				 * Do this as long as we're not in failure mode.
				 */
				if !fail {

					/*
					 * Handle overflow before multiplication.
					 */
					if vi >= LOWER_BEFORE_SHIFT {
						vi = math.MaxUint64
						fail = true
					} else {
						vi *= 10
						exp++
					}

				}

			}

			/*
			 * Check if parsing failed.
			 */
			if fail {
				return unsignedFixed{}, fmt.Errorf("%s", "Parsing failed due to arithmetic overflow.")
			} else {

				/*
				 * Create unsigned fixed-point number.
				 */
				fx := unsignedFixed{
					mantissa: vi,
					exponent: exp,
				}

				return fx, nil
			}

		}

	}

}

/*
 * Convert unsigned fixed-point number to string.
 */
func (this *unsignedFixed) String() string {
	m := this.mantissa
	digits := strconv.FormatUint(m, 10)
	numDigits := len(digits)
	e := this.exponent
	eInt := int(e)
	numLeadingZeros := (eInt - numDigits) + 1

	/*
	 * Add leading zeros if necessary.
	 */
	if numLeadingZeros > 0 {

		/*
		 * Add as many zeros as required.
		 */
		for i := 0; i < numLeadingZeros; i++ {
			digits = "0" + digits
		}

		numDigits += numLeadingZeros
	}

	dotPos := numDigits - eInt

	/*
	 * Insert dot if it is not past the last digit.
	 */
	if dotPos < numDigits {
		r := []rune(digits)
		first := r[:dotPos]
		second := r[dotPos:]
		firstString := string(first)
		secondString := string(second)
		digits = firstString + "." + secondString
	}

	return digits
}

/*
 * Checks if this unsigned fixed-point number is zero.
 */
func (this *unsignedFixed) Zero() bool {
	m := this.mantissa
	result := m == 0
	return result
}

/*
 * The distance travelled running.
 */
func (this *runningActivityStruct) DistanceKM() string {
	dist := &this.distanceKM
	s := dist.String()
	return s
}

/*
 * The duration spent running.
 */
func (this *runningActivityStruct) Duration() time.Duration {
	dur := this.duration
	return dur
}

/*
 * The energy consumed running.
 */
func (this *runningActivityStruct) EnergyKJ() uint64 {
	e := this.energyKJ
	return e
}

/*
 * The steps taken running.
 */
func (this *runningActivityStruct) StepCount() uint64 {
	steps := this.stepCount
	return steps
}

/*
 * Checks whether this is the zero value of a running activity.
 */
func (this *runningActivityStruct) Zero() bool {
	duration := this.duration
	distanceKM := this.distanceKM
	distanceKMZero := distanceKM.Zero()
	stepCount := this.stepCount
	energyKJ := this.energyKJ
	result := (duration == 0) && (distanceKMZero) && (stepCount == 0) && (energyKJ == 0)
	return result
}

/*
 * The distance travelled cycling.
 */
func (this *cyclingActivityStruct) DistanceKM() string {
	dist := &this.distanceKM
	s := dist.String()
	return s
}

/*
 * The duration spent cycling.
 */
func (this *cyclingActivityStruct) Duration() time.Duration {
	dur := this.duration
	return dur
}

/*
 * Checks whether this is the zero value of a cycling activity.
 */
func (this *cyclingActivityStruct) Zero() bool {
	duration := this.duration
	distanceKM := this.distanceKM
	distanceKMZero := distanceKM.Zero()
	energyKJ := this.energyKJ
	result := (duration == 0) && (distanceKMZero) && (energyKJ == 0)
	return result
}

/*
 * The energy consumed cycling.
 */
func (this *cyclingActivityStruct) EnergyKJ() uint64 {
	e := this.energyKJ
	return e
}

/*
 * The energy consumed performing other activities.
 */
func (this *otherActivityStruct) EnergyKJ() uint64 {
	e := this.energyKJ
	return e
}

/*
 * Checks whether this is the zero value of other activities.
 */
func (this *otherActivityStruct) Zero() bool {
	energyKJ := this.energyKJ
	result := energyKJ == 0
	return result
}

/*
 * The point in time when the activities in this group began.
 */
func (this *activityGroupStruct) Begin() time.Time {
	b := this.begin
	return b
}

/*
 * The cycling activity performed in this group.
 */
func (this *activityGroupStruct) Cycling() CyclingActivity {
	c := &this.cycling
	return c
}

/*
 * Other activities performed in this group.
 */
func (this *activityGroupStruct) Other() OtherActivity {
	o := &this.other
	return o
}

/*
 * The running activity performed in this group.
 */
func (this *activityGroupStruct) Running() RunningActivity {
	r := &this.running
	return r
}

/*
 * The weight in kilograms when the activities in this group began.
 */
func (this *activityGroupStruct) WeightKG() string {
	w := &this.weightKG
	s := w.String()
	return s
}

/*
 * Create activity group from activity info.
 */
func createActivityGroup(info *ActivityInfo) (activityGroupStruct, error) {
	errResult := error(nil)
	runningDuration := info.RunningDuration
	runningDistanceKMString := info.RunningDistanceKM
	runningDistanceKM, err := parseUnsignedFixed(runningDistanceKMString, 1)

	/*
	 * Check if this is the first error.
	 */
	if errResult == nil && err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Failed to parse running distance: %s", msg)
	}

	runningStepCount := info.RunningStepCount
	runningEnergyKJ := info.RunningEnergyKJ

	/*
	 * Create running activity.
	 */
	runningActivity := runningActivityStruct{
		duration:   runningDuration,
		distanceKM: runningDistanceKM,
		stepCount:  runningStepCount,
		energyKJ:   runningEnergyKJ,
	}

	cyclingDuration := info.CyclingDuration
	cyclingDistanceKMString := info.CyclingDistanceKM
	cyclingDistanceKM, err := parseUnsignedFixed(cyclingDistanceKMString, 1)

	/*
	 * Check if this is the first error.
	 */
	if errResult == nil && err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Failed to parse cycling distance: %s", msg)
	}

	cyclingEnergyKJ := info.CyclingEnergyKJ

	/*
	 * Create cycling activity.
	 */
	cyclingActivity := cyclingActivityStruct{
		duration:   cyclingDuration,
		distanceKM: cyclingDistanceKM,
		energyKJ:   cyclingEnergyKJ,
	}

	otherEnergyKJ := info.OtherEnergyKJ

	/*
	 * Create other activity.
	 */
	otherActivity := otherActivityStruct{
		energyKJ: otherEnergyKJ,
	}

	begin := info.Begin
	weightKGString := info.WeightKG
	weightKG, err := parseUnsignedFixed(weightKGString, 1)

	/*
	 * Check if this is the first error.
	 */
	if errResult == nil && err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Failed to parse weight: %s", msg)
	}

	/*
	 * Create activity group.
	 */
	g := activityGroupStruct{
		begin:    begin,
		weightKG: weightKG,
		running:  runningActivity,
		cycling:  cyclingActivity,
		other:    otherActivity,
	}

	return g, errResult
}

/*
 * Search an activity group by its beginning time.
 *
 * If an activity group with this exact beginning time exists, the first return
 * value is the index of the activity group within the groups slice and the
 * second return value is true.
 *
 * If an activity group with this exact beginning time does not exist, the
 * first return value is the index of the first activity group that starts
 * later, i. e. the index at which an activity group with the given beginning
 * time shall be inserted into the groups slice, and the second return value is
 * false.
 */
func (this *activitiesStruct) searchActivity(begin time.Time) (uint64, bool) {
	groups := this.groups
	numGroups := len(groups)
	numGroups64 := uint64(numGroups)
	idxLeft := uint64(0)

	/*
	 * The case that there are no groups requires special handling.
	 */
	if numGroups64 > 0 {
		idxRight := numGroups64 - 1

		/*
		 * Binary search algorithm.
		 */
		for idxLeft <= idxRight {
			diff := idxRight - idxLeft
			diffHalf := diff >> 1
			idxPivot := idxLeft + diffHalf
			pivot := groups[idxPivot]
			pivotBegin := pivot.begin

			/*
			 * Check if we have to continue searching in left or
			 * right half.
			 */
			if pivotBegin.After(begin) {
				idxRight = idxPivot - 1
			} else if pivotBegin.Before(begin) {
				idxLeft = idxPivot + 1
			} else {
				return idxPivot, true
			}

		}

	}

	return idxLeft, false
}

/*
 * Insert new activities.
 */
func (this *activitiesStruct) Add(info *ActivityInfo) error {
	g, err := createActivityGroup(info)

	/*
	 * Only try to insert activity group, if there were no errors so far.
	 *
	 * Insert the newly created group into the right place.
	 *
	 * All activity groups shall be ordered by the time of their beginning
	 * in UTC.
	 */
	if err == nil {
		this.mutex.Lock()
		groups := this.groups
		numGroups := len(groups)

		/*
		 * Limit number of groups.
		 */
		if numGroups > math.MaxUint32 {
			err = fmt.Errorf("There cannot be more than %d activity groups.", math.MaxUint32)
		} else {
			begin := info.Begin
			beginUTC := begin.UTC()
			idxInsert, exists := this.searchActivity(beginUTC)

			/*
			 * Insert new activity group if no group with this
			 * beginning exists.
			 */
			if exists {
				err = fmt.Errorf("%s", "Activity group with this beginning already exists.")
			} else {
				empty := activityGroupStruct{}
				groups = append(groups, empty)
				idxInsertInc := idxInsert + 1
				copy(groups[idxInsertInc:], groups[idxInsert:])
				groups[idxInsert] = g
				this.groups = groups
				this.revision++
			}

		}

		this.mutex.Unlock()
	}

	return err
}

/*
 * Determine the time when a certain activity ends.
 *
 * If there is a subsequent activity, this is the point in time when that
 * subsequent activity starts.
 *
 * If there is no subsequent activity, this is one day after the point in time
 * when the activity started.
 */
func (this *activitiesStruct) End(id uint32) (time.Time, error) {
	this.mutex.RLock()
	groups := this.groups
	numGroups := len(groups)
	numGroups64 := uint64(numGroups)
	lastGroupIdx := numGroups64 - 1
	id64 := uint64(id)

	/*
	 * Check if group exists or is the last group.
	 */
	if id64 >= numGroups64 {
		this.mutex.RUnlock()
		return time.Time{}, fmt.Errorf("No activity group with index %d. (There are %d activity groups.)", id64, numGroups64)
	} else if id64 == lastGroupIdx {
		g := groups[lastGroupIdx]
		this.mutex.RUnlock()
		timeLast := g.begin
		result := timeLast.Add(TIME_DAY)
		return result, nil
	} else {
		idInc64 := id64 + 1
		g := groups[idInc64]
		this.mutex.RUnlock()
		result := g.begin
		return result, nil
	}

}

/*
 * Serialize activities to JSON structure.
 */
func (this *activitiesStruct) Export() ([]byte, error) {
	this.mutex.RLock()
	groups := this.groups
	numGroups := len(groups)
	infos := make([]ActivityInfo, numGroups)

	/*
	 * Iterate over all activity groups.
	 */
	for idx, g := range groups {
		begin := g.Begin()
		weightKG := g.WeightKG()
		running := g.Running()
		runningDuration := running.Duration()
		runningDistanceKM := running.DistanceKM()
		runningStepCount := running.StepCount()
		runningEnergyKJ := running.EnergyKJ()
		cycling := g.Cycling()
		cyclingDuration := cycling.Duration()
		cyclingDistanceKM := cycling.DistanceKM()
		cyclingEnergyKJ := cycling.EnergyKJ()
		other := g.Other()
		otherEnergyKJ := other.EnergyKJ()

		/*
		 * Create activity info.
		 */
		info := ActivityInfo{
			Begin:             begin,
			WeightKG:          weightKG,
			RunningDuration:   runningDuration,
			RunningDistanceKM: runningDistanceKM,
			RunningStepCount:  runningStepCount,
			RunningEnergyKJ:   runningEnergyKJ,
			CyclingDuration:   cyclingDuration,
			CyclingDistanceKM: cyclingDistanceKM,
			CyclingEnergyKJ:   cyclingEnergyKJ,
			OtherEnergyKJ:     otherEnergyKJ,
		}

		infos[idx] = info
	}

	buf, err := json.MarshalIndent(infos, "", "\t")

	/*
	 * Check if error occured during serialization.
	 */
	if err != nil {
		this.mutex.RUnlock()
		msg := err.Error()
		return nil, fmt.Errorf("Activity data serialization failed: %s", msg)
	} else {
		this.mutex.RUnlock()
		return buf, nil
	}

}

/*
 * Obtain a certain activity group.
 */
func (this *activitiesStruct) Get(id uint32) (ActivityGroup, error) {
	this.mutex.RLock()
	groups := this.groups
	numGroups := len(groups)
	numGroups64 := uint64(numGroups)
	id64 := uint64(id)

	/*
	 * Check if group exists.
	 */
	if id64 >= numGroups64 {
		this.mutex.RUnlock()
		return nil, fmt.Errorf("Activity group with id %d does not exist. There are only %d groups.", id, numGroups)
	} else {
		g := groups[id]
		this.mutex.RUnlock()
		return &g, nil
	}

}

/*
 * Import activities from JSON.
 */
func (this *activitiesStruct) Import(buf []byte) error {
	infos := []ActivityInfo{}
	err := json.Unmarshal(buf, &infos)

	/*
	 * Check if data could be deserialized.
	 */
	if err != nil {
		msg := err.Error()
		return fmt.Errorf("Error deserializing activity data: %s", msg)
	} else {
		this.mutex.Lock()
		groups := this.groups
		firstError := error(nil)
		idxFirstErr := uint64(0)
		numErrors := uint64(0)

		/*
		 * Iterate over activity infos.
		 */
		for idx, info := range infos {
			g, err := createActivityGroup(&info)

			/*
			 * Check if activity group could be parsed.
			 */
			if err != nil {

				/*
				 * Store first error occuring.
				 */
				if firstError == nil {
					firstError = err
					idxFirstErr = uint64(idx)
				}

				/*
				 * Increment error count.
				 */
				if numErrors < math.MaxUint64 {
					numErrors++
				}

			} else {
				groups = append(groups, g)
			}

		}

		/*
		 * Comparison function for sorting algorithm.
		 */
		less := func(i int, j int) bool {
			gi := groups[i]
			giBegin := gi.begin
			gj := groups[j]
			gjBegin := gj.begin
			result := giBegin.Before(gjBegin)
			return result
		}

		sort.SliceStable(groups, less)
		this.groups = groups
		this.revision++
		this.mutex.Unlock()

		/*
		 * Check if error occured.
		 */
		if firstError != nil {
			msg := firstError.Error()
			return fmt.Errorf("Error deserializing activity data: %d erroneous activity groups, first at group number %d: %s", numErrors, idxFirstErr, msg)
		} else {
			return nil
		}

	}

}

/*
 * Import activities from CSV.
 */
func (this *activitiesStruct) ImportCSV(data string) error {
	rstr := strings.NewReader(data)
	rcsv := csv.NewReader(rstr)
	records, err := rcsv.ReadAll()

	/*
	 * Check if an error occured.
	 */
	if err != nil {
		msg := err.Error()
		return fmt.Errorf("Error importing activity data from CSV: %s", msg)
	} else {
		this.mutex.Lock()
		groups := this.groups
		numGroups := len(groups)
		groupsCopy := make([]activityGroupStruct, numGroups)
		copy(groupsCopy, groups)
		firstError := error(nil)
		idxFirstErr := uint64(0)
		numErrors := uint64(0)

		/*
		 * Iterate over all records and parse activity data.
		 */
		for idx, record := range records {
			recordHasErrors := false
			numFields := len(record)

			/*
			 * Check that sufficient number of fields is present.
			 */
			if numFields < EXPECTED_NUM_FIELDS {

				/*
				 * Store first error occuring.
				 */
				if firstError == nil {
					firstError = fmt.Errorf("Expected %d fields, found %d.", EXPECTED_NUM_FIELDS, numFields)
					idxFirstErr = uint64(idx)
				}

				/*
				 * Increment error count.
				 */
				if !recordHasErrors && numErrors < math.MaxUint64 {
					numErrors++
					recordHasErrors = true
				}

			} else {
				beginString := record[0]
				begin, err := filter.ParseTime(beginString, false)

				/*
				 * Check if begin time could be parsed.
				 */
				if err != nil {

					/*
					 * Store first error occuring.
					 */
					if firstError == nil {
						msg := err.Error()
						firstError = fmt.Errorf("Failed to parse begin time stamp: %s", msg)
						idxFirstErr = uint64(idx)
					}

					/*
					 * Increment error count.
					 */
					if !recordHasErrors && numErrors < math.MaxUint64 {
						numErrors++
						recordHasErrors = true
					}

				}

				weightKG := record[1]

				/*
				 * Allow for empty weight.
				 */
				if weightKG == "" {
					weightKG = "0.0"
				}

				runningDurationString := record[2]
				runningDuration := time.Duration(0)

				/*
				 * Allow for empty running duration.
				 */
				if runningDurationString != "" {
					runningDuration, err = time.ParseDuration(runningDurationString)

					/*
					 * Check if running duration could be parsed.
					 */
					if err != nil {

						/*
						 * Store first error occuring.
						 */
						if firstError == nil {
							msg := err.Error()
							firstError = fmt.Errorf("Failed to parse running duration: %s", msg)
							idxFirstErr = uint64(idx)
						}

						/*
						 * Increment error count.
						 */
						if !recordHasErrors && numErrors < math.MaxUint64 {
							numErrors++
							recordHasErrors = true
						}

					}

				}

				runningDistanceKM := record[3]

				/*
				 * Allow for empty running distance.
				 */
				if runningDistanceKM == "" {
					runningDistanceKM = "0.0"
				}

				runningStepCountString := record[4]
				runningStepCount := uint64(0)

				/*
				 * Allow for empty running step count.
				 */
				if runningStepCountString != "" {
					runningStepCount, err = strconv.ParseUint(runningStepCountString, 10, 64)

					/*
					 * Check if running step count could be parsed.
					 */
					if err != nil {

						/*
						 * Store first error occuring.
						 */
						if firstError == nil {
							msg := err.Error()
							firstError = fmt.Errorf("Failed to parse running step count: %s", msg)
							idxFirstErr = uint64(idx)
						}

						/*
						 * Increment error count.
						 */
						if !recordHasErrors && numErrors < math.MaxUint64 {
							numErrors++
							recordHasErrors = true
						}

					}

				}

				runningEnergyKJString := record[5]
				runningEnergyKJ := uint64(0)

				/*
				 * Allow for empty running energy.
				 */
				if runningEnergyKJString != "" {
					runningEnergyKJ, err = strconv.ParseUint(runningEnergyKJString, 10, 64)

					/*
					 * Check if running energy could be parsed.
					 */
					if err != nil {

						/*
						 * Store first error occuring.
						 */
						if firstError == nil {
							msg := err.Error()
							firstError = fmt.Errorf("Failed to parse running energy: %s", msg)
							idxFirstErr = uint64(idx)
						}

						/*
						 * Increment error count.
						 */
						if !recordHasErrors && numErrors < math.MaxUint64 {
							numErrors++
							recordHasErrors = true
						}

					}

				}

				cyclingDurationString := record[6]
				cyclingDuration := time.Duration(0)

				/*
				 * Allow for empty cycling duration.
				 */
				if cyclingDurationString != "" {
					cyclingDuration, err = time.ParseDuration(cyclingDurationString)

					/*
					 * Check if cycling duration could be parsed.
					 */
					if err != nil {

						/*
						 * Store first error occuring.
						 */
						if firstError == nil {
							msg := err.Error()
							firstError = fmt.Errorf("Failed to parse cycling duration: %s", msg)
							idxFirstErr = uint64(idx)
						}

						/*
						 * Increment error count.
						 */
						if !recordHasErrors && numErrors < math.MaxUint64 {
							numErrors++
							recordHasErrors = true
						}

					}

				}

				cyclingDistanceKM := record[7]

				/*
				 * Allow for empty cycling distance.
				 */
				if cyclingDistanceKM == "" {
					cyclingDistanceKM = "0.0"
				}

				cyclingEnergyKJString := record[8]
				cyclingEnergyKJ := uint64(0)

				/*
				 * Allow for empty cycling energy.
				 */
				if cyclingEnergyKJString != "" {
					cyclingEnergyKJ, err = strconv.ParseUint(cyclingEnergyKJString, 10, 64)

					/*
					 * Check if cycling energy could be parsed.
					 */
					if err != nil {

						/*
						 * Store first error occuring.
						 */
						if firstError == nil {
							msg := err.Error()
							firstError = fmt.Errorf("Failed to parse cycling energy: %s", msg)
							idxFirstErr = uint64(idx)
						}

						/*
						 * Increment error count.
						 */
						if !recordHasErrors && numErrors < math.MaxUint64 {
							numErrors++
							recordHasErrors = true
						}

					}

				}

				otherEnergyKJString := record[9]
				otherEnergyKJ := uint64(0)

				/*
				 * Allow for empty other energy.
				 */
				if otherEnergyKJString != "" {
					otherEnergyKJ, err = strconv.ParseUint(otherEnergyKJString, 10, 64)

					/*
					 * Check if other energy could be parsed.
					 */
					if err != nil {

						/*
						 * Store first error occuring.
						 */
						if firstError == nil {
							msg := err.Error()
							firstError = fmt.Errorf("Failed to parse other energy: %s", msg)
							idxFirstErr = uint64(idx)
						}

						/*
						 * Increment error count.
						 */
						if !recordHasErrors && numErrors < math.MaxUint64 {
							numErrors++
							recordHasErrors = true
						}

					}

				}

				/*
				 * Create activity info.
				 */
				info := ActivityInfo{
					Begin:             begin,
					WeightKG:          weightKG,
					RunningDuration:   runningDuration,
					RunningDistanceKM: runningDistanceKM,
					RunningStepCount:  runningStepCount,
					RunningEnergyKJ:   runningEnergyKJ,
					CyclingDuration:   cyclingDuration,
					CyclingDistanceKM: cyclingDistanceKM,
					CyclingEnergyKJ:   cyclingEnergyKJ,
					OtherEnergyKJ:     otherEnergyKJ,
				}

				g, err := createActivityGroup(&info)

				/*
				 * Check if activity group could be parsed.
				 */
				if err != nil {

					/*
					 * Store first error occuring.
					 */
					if firstError == nil {
						firstError = err
						idxFirstErr = uint64(idx)
					}

					/*
					 * Increment error count.
					 */
					if !recordHasErrors && numErrors < math.MaxUint64 {
						numErrors++
					}

				} else {
					groupsCopy = append(groupsCopy, g)
				}

			}

		}

		/*
		 * Only modify activity groups if no error occured.
		 */
		if firstError == nil {

			/*
			 * Comparison function for sorting algorithm.
			 */
			less := func(i int, j int) bool {
				gi := groupsCopy[i]
				giBegin := gi.begin
				gj := groupsCopy[j]
				gjBegin := gj.begin
				result := giBegin.Before(gjBegin)
				return result
			}

			sort.SliceStable(groupsCopy, less)
			this.groups = groupsCopy
			this.revision++
		}

		this.mutex.Unlock()

		/*
		 * Check if error occured.
		 */
		if firstError != nil {
			msg := firstError.Error()
			return fmt.Errorf("Error deserializing activity data: %d erroneous activity groups, first at group number %d: %s", numErrors, idxFirstErr, msg)
		} else {
			return nil
		}

	}

}

/*
 * Determine the number of activity groups.
 */
func (this *activitiesStruct) Length() uint32 {
	this.mutex.RLock()
	groups := this.groups
	length := len(groups)
	this.mutex.RUnlock()
	length32 := uint32(length)
	return length32
}

/*
 * Removes an activity group.
 */
func (this *activitiesStruct) Remove(id uint32) error {
	err := error(nil)
	this.mutex.Lock()
	groups := this.groups
	length := len(groups)
	length64 := uint64(length)
	id64 := uint64(id)

	/*
	 * Check if activity group exists.
	 */
	if id64 >= length64 {
		err = fmt.Errorf("No activity group with id = %d.", id64)
	} else {
		idInc64 := id64 + 1
		groups = append(groups[:id64], groups[idInc64:]...)
		this.groups = groups
		this.revision++
	}

	this.mutex.Unlock()
	return err
}

/*
 * Replaces an activity group with a newly created one.
 */
func (this *activitiesStruct) Replace(id uint32, info *ActivityInfo) error {
	g, err := createActivityGroup(info)

	/*
	 * Only try to replace activity group, if there were no errors so far.
	 *
	 * Replace the specified group with the newly created one, then sort
	 * all activitiy groups by the time of their beginning in UTC.
	 */
	if err == nil {
		this.mutex.Lock()
		groups := this.groups
		length := len(groups)
		length64 := uint64(length)
		id64 := uint64(id)

		/*
		 * Check if activity group exists.
		 */
		if id64 >= length64 {
			err = fmt.Errorf("No activity group with id = %d.", id64)
		} else {
			groups[id64] = g

			/*
			 * Comparison function for sorting algorithm.
			 */
			less := func(i int, j int) bool {
				gi := groups[i]
				giBegin := gi.begin
				gj := groups[j]
				gjBegin := gj.begin
				result := giBegin.Before(gjBegin)
				return result
			}

			sort.SliceStable(groups, less)
			this.revision++
		}

		this.mutex.Unlock()
	}

	return err
}

/*
 * Returns the current revision number.
 *
 * This number is incremented each time activity information is modified.
 */
func (this *activitiesStruct) Revision() uint64 {
	this.mutex.RLock()
	rev := this.revision
	this.mutex.RUnlock()
	return rev
}

/*
 * Create data structure storing activities.
 */
func CreateActivities() Activities {
	g := []activityGroupStruct{}

	/*
	 * Create activity storage.
	 */
	a := activitiesStruct{
		groups:   g,
		revision: 0,
	}

	return &a
}
