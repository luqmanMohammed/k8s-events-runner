package configcollector

type ConfigCollector interface {
	Collect() error
}
