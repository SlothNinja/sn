package sn

func toSentence(strings []string) (sentence string) {
	switch length := len(strings); length {
	case 0:
	case 1:
		sentence = strings[0]
	case 2:
		sentence = strings[0] + " and " + strings[1]
	default:
		for i, s := range strings {
			switch i {
			case 0:
				sentence += s
			case length - 1:
				sentence += ", and " + s
			default:
				sentence += ", " + s
			}
		}
	}
	return sentence
}
