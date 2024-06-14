package image_processing

import (
	"go-steg/cli/helpers"
	"testing"
)

func TestMultiCarrierDecodeByFileNames(t *testing.T) {
	type args struct {
		carrierFileNames []string
		password         string
		outputFileDir    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test MultiCarrierDecodeByFileNames",
			args: args{
				carrierFileNames: []string{"../../go_steg/pics/testPhotoOutput/carrierPhoto-0-embedded.png"},
				password:         "testPassword",
				outputFileDir:    "../../go_steg/pics/testPhotoOutput",
			},
		},
	}
	for _, tt := range tests {
		helpers.UseMask = true
		t.Run(tt.name, func(t *testing.T) {
			if err := MultiCarrierDecodeByFileNames(tt.args.carrierFileNames, tt.args.password, tt.args.outputFileDir); (err != nil) != tt.wantErr {
				t.Errorf("MultiCarrierDecodeByFileNames() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
