package cmd

/*
Copyright © 2023 Judson Stevens oss@judsonstevens.dev

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

import (
	"fmt"
	"go-steg/app/image_processing"

	"github.com/spf13/cobra"
)

var embedFileName string
var carrierFileNames []string
var password string

// encodeCmd represents the encode command
var encodeCmd = &cobra.Command{
	Use:   "encode",
	Short: "Embed a photo into another photo or group of photos",
	Long: `Given an "embed" photo, a single or list of "carrier" photos, and a password, 
embed the "embed" photo into the "carrier" photo(s) and output the resulting altered files with the mask information. 
This method will use the passed in password to attempt to generate a mask to use to secure the embedded information.
Depending on the mask generated, we may need to generate a new mask to use to secure the embedded information,
as the size of embed information may be larger than the mask can handle.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("encode called")
		fmt.Println("embed-file-name:", embedFileName)
		fmt.Println("carrierFileName:", carrierFileNames)
		fmt.Println("password:", password)
		err := image_processing.EncodeByFileNames(
			carrierFileNames,
			embedFileName,
			1)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(encodeCmd)

	encodeCmd.PersistentFlags().StringVarP(
		&embedFileName,
		"embedFileName",
		"e",
		"",
		"The name of the file to embed into the carrier file(s)")
	err := encodeCmd.MarkPersistentFlagRequired("embedFileName")
	if err != nil {
		panic(err)
	}

	// TODO: Handle names being passed in that aren't files - that is, check to see that they exist and are parsable/the right format
	encodeCmd.PersistentFlags().StringSliceVarP(
		&carrierFileNames,
		"carrierFileNames",
		"c",
		[]string{},
		"A single name, or a comma separate list of names, of the carrier file(s) to embed the embed file into")
	err = encodeCmd.MarkPersistentFlagRequired("carrierFileNames")
	if err != nil {
		panic(err)
	}

	encodeCmd.PersistentFlags().StringVarP(
		&password,
		"password",
		"p",
		"",
		"The password to use to generate the mask to use to embed the embed file into the carrier file(s)")
}
