package fizz

// config holds the settings built from the Options passed
// to New and NewFromEngine. It is empty until the options land.
type config struct{}

// Option configures a Fizz instance. Options are passed to
// New and NewFromEngine and applied in order; nil options are ignored.
type Option func(*config)
