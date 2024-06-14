package image_processing

import (
	"go-steg/cli/helpers"
	"testing"
)

func TestEncodeByFileNames(t *testing.T) {
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
			name: "Test EncodeByFileNames",
			args: args{
				carrierFileNames: []string{"../../go_steg/pics/carrierPhoto.png"},
				dataFileName:     "../../go_steg/pics/embedPhoto.png",
				uniquePhotoID:    1,
				password:         "testPassword",
				outputFileDir:    "../../go_steg/pics/testPhotoOutput",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.UseMask = true
			if err := EncodeByFileNames(tt.args.carrierFileNames, tt.args.dataFileName, tt.args.uniquePhotoID, tt.args.password, tt.args.outputFileDir); (err != nil) != tt.wantErr {
				t.Errorf("EncodeByFileNames() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
