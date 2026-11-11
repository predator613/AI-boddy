package services

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reDayMonth = regexp.MustCompile(`(\d{1,2})\s+(январ|феврал|март|апрел|ма[яй]|июн|июл|август|сентябр|октябр|ноябр|декабр)`)
	reNumeric  = regexp.MustCompile(`\b(\d{1,2})\.(\d{1,2})(?:\.(\d{2,4}))?\b`)
	reWeekday  = regexp.MustCompile(`понедельник|вторник|сред[уа]|четверг|пятниц[уа]|суббот[уа]|воскресень[ея]`)
	monthStems = []string{"январ", "феврал", "март", "апрел", "ма", "июн", "июл", "август", "сентябр", "октябр", "ноябр", "декабр"}
	dayStems   = map[string]time.Weekday{"понедельник": time.Monday, "вторник": time.Tuesday, "сред": time.Wednesday,
		"четверг": time.Thursday, "пятниц": time.Friday, "суббот": time.Saturday, "воскресень": time.Sunday}
)

// ParseLeaveDate ищет дату в тексте: «14 ноября», «14.11», «завтра», «послезавтра», «в пятницу».
// Дата без года берется ближайшая будущая. ok = false – даты в тексте нет.
func ParseLeaveDate(text string, today time.Time) (time.Time, bool) {
	t := strings.ToLower(text)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	switch {
	case strings.Contains(t, "послезавтра"):
		return today.AddDate(0, 0, 2), true
	case strings.Contains(t, "завтра"):
		return today.AddDate(0, 0, 1), true
	}
	if m := reDayMonth.FindStringSubmatch(t); m != nil {
		day, _ := strconv.Atoi(m[1])
		for i, stem := range monthStems {
			if strings.HasPrefix(m[2], stem) {
				return future(today, time.Month(i+1), day)
			}
		}
	}
	if m := reNumeric.FindStringSubmatch(t); m != nil {
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		if month < 1 || month > 12 {
			return time.Time{}, false
		}
		if m[3] != "" {
			year, _ := strconv.Atoi(m[3])
			if year < 100 {
				year += 2000
			}
			d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, today.Location())
			return d, d.Day() == day
		}
		return future(today, time.Month(month), day)
	}
	if m := reWeekday.FindString(t); m != "" {
		for stem, wd := range dayStems {
			if strings.HasPrefix(m, stem) {
				diff := (int(wd) - int(today.Weekday()) + 7) % 7
				if diff == 0 {
					diff = 7
				}
				return today.AddDate(0, 0, diff), true
			}
		}
	}
	return time.Time{}, false
}

func future(today time.Time, month time.Month, day int) (time.Time, bool) {
	d := time.Date(today.Year(), month, day, 0, 0, 0, 0, today.Location())
	if d.Day() != day {
		return time.Time{}, false
	}
	if d.Before(today) {
		d = d.AddDate(1, 0, 0)
	}
	return d, true
}

// AddWorkdays – дата через n рабочих дней (праздники не учитываются).
func AddWorkdays(from time.Time, n int) time.Time {
	d := from
	for n > 0 {
		d = d.AddDate(0, 0, 1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			n--
		}
	}
	return d
}

// WorkdaysBetween – сколько рабочих дней от from (не включая) до to (включая).
func WorkdaysBetween(from, to time.Time) int {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	n := 0
	for d := from.AddDate(0, 0, 1); !d.After(to); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			n++
		}
	}
	return n
}

// IsWeekend – суббота или воскресенье.
func IsWeekend(d time.Time) bool { return d.Weekday() == time.Saturday || d.Weekday() == time.Sunday }
