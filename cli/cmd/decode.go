package cmd

/*
Copyright © 2023 Judson Stevens oss@judsonstevens.dev

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
with the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS," WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

import (
	"go-steg/cli/helpers"
	"go-steg/go_steg/image_processing"

	"github.com/spf13/cobra"
)

var decodeCarrierFileNames []string
var decodePassword string
var decodeOutputFileDir string

// decodeCmd represents the decode command
var decodeCmd = &cobra.Command{
	Use:   "decode",
	Short: "Decode a single or multiple carrier photos to produce the embed photo",
	Long: `Given one more more "carrier" photos and a password, decode the hidden information in
the carrier photos to produce the "embed" photo. The password will be used to regenerate the mask
and decode the information from the carrier photos.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := helpers.ValidateIsValidDirectory(outputFileDir)
		if err != nil {
			panic(err)
		}

		for _, fileName := range decodeCarrierFileNames {
			err := helpers.ValidateIsValidFile(fileName)
			if err != nil {
				panic(err)
			}
		}

		err = image_processing.MultiCarrierDecodeByFileNames(decodeCarrierFileNames, decodePassword, decodeOutputFileDir)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(decodeCmd)

	decodeCmd.PersistentFlags().StringSliceVarP(
		&decodeCarrierFileNames,
		"carrierFileNames",
		"c",
		[]string{},
		"List of carrier file names to decode. Please use either a valid relative path or a valid absolute path")
	err := decodeCmd.MarkPersistentFlagRequired("carrierFileNames")
	if err != nil {
		panic(err)
	}

	decodeCmd.PersistentFlags().StringVarP(
		&decodePassword,
		"password",
		"p",
		"",
		"Password to use to decode the carrier photos")
	err = decodeCmd.MarkPersistentFlagRequired("password")
	if err != nil {
		panic(err)
	}

	encodeCmd.PersistentFlags().StringVarP(
		&decodeOutputFileDir,
		"outputFileDir",
		"o",
		"",
		"The directory to output the resulting files to. This must be either a valid relative path or "+
			"a valid absolute path")
	err = encodeCmd.MarkPersistentFlagRequired("outputFileDir")
	if err != nil {
		panic(err)
	}

	err = encodeCmd.PersistentFlags().SetAnnotation("outputFileDir", cobra.BashCompSubdirsInDir, []string{})
	if err != nil {
		panic(err)
	}
}
