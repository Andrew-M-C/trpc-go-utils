package plugin

type options struct {
	dependsOn     []string
	flexDependsOn []string
}

type Option func(*options)

func mergeOptions(opts []Option) options {
	opt := options{}
	for _, o := range opts {
		if o != nil {
			o(&opt)
		}
	}
	return opt
}

// WithDependsOn 指定 plugin 的强依赖关系
func WithDependsOn(deps ...string) Option {
	return func(o *options) {
		o.dependsOn = deps
	}
}

// WithFlexDependsOn 指定 plugin 的弱依赖关系
func WithFlexDependsOn(deps ...string) Option {
	return func(o *options) {
		o.flexDependsOn = deps
	}
}
