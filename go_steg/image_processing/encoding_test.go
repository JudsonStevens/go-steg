package image_processing

import "testing"

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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := EncodeByFileNames(tt.args.carrierFileNames, tt.args.dataFileName, tt.args.uniquePhotoID, tt.args.password, tt.args.outputFileDir); (err != nil) != tt.wantErr {
				t.Errorf("EncodeByFileNames() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
