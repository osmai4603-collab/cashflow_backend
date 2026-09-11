package calendar

type AppointmentSlot struct {
	ID                int64   `json:"id"`
	AppointmentTypeID int64   `json:"appointment_type_id"`
	DayOfWeek         int     `json:"day_of_week"`
	HourFrom          float64 `json:"hour_from"`
	HourTo            float64 `json:"hour_to"`
}

func (slot *AppointmentSlot) Validate() error {
	if slot.AppointmentTypeID <= 0 || slot.DayOfWeek < 0 || slot.DayOfWeek > 6 || slot.HourFrom < 0 || slot.HourTo > 24 || slot.HourTo <= slot.HourFrom {
		return invalidCalendar("appointment slot has invalid day or hours")
	}
	return nil
}
