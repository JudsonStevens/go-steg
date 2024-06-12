package image_processing

import "testing"

func TestMultiCarrierEncodeByFileNames(t *testing.T) {
	type args struct {
		carrierFileNames []string
		dataFileName     string
		uniquePhotoID    uint64
		password         string
		outputFileDir    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test MultiCarrierEncodeByFileNames is Successful",
			args: args{
				carrierFileNames: []string{"../test_files/carrierPhoto.png"},
				dataFileName:     "../test_files/embedPhoto.png",
				uniquePhotoID:    1,
				password:         "password",
				outputFileDir:    "../test_files",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MultiCarrierEncodeByFileNames(tt.args.carrierFileNames, tt.args.dataFileName, tt.args.uniquePhotoID, tt.args.password, tt.args.outputFileDir); (err != nil) != tt.wantErr {
				t.Errorf("MultiCarrierEncodeByFileNames() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
