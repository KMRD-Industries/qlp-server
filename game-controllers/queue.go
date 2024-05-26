package game_controllers

type Queue struct {
	elements []Coordinate
}

func (q *Queue) put(element Coordinate) {
	q.elements = append(q.elements, element)
}

func (q *Queue) get() (Coordinate, bool) {
	if q.isEmpty() {
		return Coordinate{0, 0}, false // kolejka jest pusta więc zwracamy false - nie udało się pobrać z kolejki
	}
	value := q.elements[0]
	q.elements = q.elements[1:]
	return value, true
}

func (q *Queue) isEmpty() bool {
	if len(q.elements) == 0 {
		return true
	}
	return false
}
