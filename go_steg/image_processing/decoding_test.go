package image_processing

import "testing"

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
			name: "Test MultiCarrierDecodeByFileNames is Successful",
			args: args{
				carrierFileNames: []string{"../test_files/carrierPhoto-0-embedded.png"},
				password:         "password",
				outputFileDir:    "../test_files",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MultiCarrierDecodeByFileNames(tt.args.carrierFileNames, tt.args.password, tt.args.outputFileDir); (err != nil) != tt.wantErr {
				t.Errorf("MultiCarrierDecodeByFileNames() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
