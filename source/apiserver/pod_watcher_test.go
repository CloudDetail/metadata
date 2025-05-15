package apiserver

import "testing"

func Test_cutContainerId(t *testing.T) {
	type args struct {
		containerIdStatus string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "containerd runtime",
			args: args{
				containerIdStatus: "containerd://286b025a9464cb948a3f388df8a6700895fab34ff01d4770d308c6ae00508c8d", // "containerd://286b025a9464cb948a3f388df8a6700895fab34ff01d4770d308c6ae00508c8d"
			},
			want: "286b025a9464",
		},
		{
			name: "docker runtime",
			args: args{
				containerIdStatus: "docker://be17c1ed0a0b385a5d7735dc7bbd989662cbe820f6ec6ead7c3458cafb3309cc", // "containerd://286b025a9464cb948a3f388df8a6700895fab34ff01d4770d308c6ae00508c8d"
			},
			want: "be17c1ed0a0b",
		},
		{
			name: "cri-o runtime",
			args: args{
				containerIdStatus: "cri-o://ce50acc6dddbd9efcc5b4005adffe4ec63d38a18d365b6c7af20d2b97a443f41",
			},
			want: "ce50acc6dddb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cutContainerId(tt.args.containerIdStatus); got != tt.want {
				t.Errorf("cutContainerId() = %v, want %v", got, tt.want)
			}
		})
	}
}
