package oracle

type Oracle struct{}

func NewOracle() *Oracle {
	return &Oracle{}
}

func (o *Oracle) Ask(question string) string {
	return "42"
}
