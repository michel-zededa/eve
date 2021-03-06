// Copyright (c) 2018-2019 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package tpmmgr

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"reflect"
	"time"
	"unsafe"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpmutil"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/pkg/pillar/agentlog"
	"github.com/lf-edge/eve/pkg/pillar/pidfile"
	"github.com/lf-edge/eve/pkg/pillar/pubsub"
	pubsublegacy "github.com/lf-edge/eve/pkg/pillar/pubsub/legacy"
	"github.com/lf-edge/eve/pkg/pillar/types"
	log "github.com/sirupsen/logrus"
)

type tpmMgrContext struct {
	subGlobalConfig pubsub.Subscription
	GCInitialized   bool // GlobalConfig initialized
}

const (
	agentName = "tpmmgr"
	//TpmPubKeyName is the file to store TPM public key file
	TpmPubKeyName = "/var/tmp/tpm.eccpubk.der"

	//TpmDeviceCertFileName is the file name to store device certificate
	TpmDeviceCertFileName = types.DeviceCertName

	//TpmDevicePath is the TPM device file path
	TpmDevicePath = "/dev/tpmrm0"

	//TpmDeviceKeyHdl is the well known TPM permanent handle for device key
	TpmDeviceKeyHdl tpmutil.Handle = 0x817FFFFF

	//TpmEKHdl is the well known TPM permanent handle for Endorsement key
	TpmEKHdl tpmutil.Handle = 0x81000001

	//TpmSRKHdl is the well known TPM permanent handle for Storage key
	TpmSRKHdl tpmutil.Handle = 0x81000002

	//TpmAKHdl is the well known TPM permanent handle for AIK key
	TpmAKHdl tpmutil.Handle = 0x81000003

	//TpmDeviceCertHdl is the well known TPM NVIndex for device cert
	TpmDeviceCertHdl tpmutil.Handle = 0x1500000

	//TpmPasswdHdl is the well known TPM NVIndex for TPM Credentials
	TpmPasswdHdl tpmutil.Handle = 0x1600000

	//TpmDiskKeyHdl is the handle for constructing disk encryption key
	TpmDiskKeyHdl tpmutil.Handle = 0x1700000

	tpmCredentialsFileName = types.IdentityDirname + "/tpm_credential"
	emptyPassword          = ""
	tpmLockName            = types.TmpDirname + "/tpm.lock"
	maxPasswdLength        = 7  //limit TPM password to this length
	vaultKeyLength         = 32 //Bytes

	// Time limits for event loop handlers
	errorTime   = 3 * time.Minute
	warningTime = 40 * time.Second
)

var (
	tpmHwInfo        = ""
	pcrSelection     = tpm2.PCRSelection{Hash: tpm2.AlgSHA1, PCRs: []int{7}}
	pcrListForQuote  = tpm2.PCRSelection{Hash: tpm2.AlgSHA256, PCRs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8}}
	defaultKeyParams = tpm2.Public{
		Type:    tpm2.AlgECC,
		NameAlg: tpm2.AlgSHA256,
		Attributes: tpm2.FlagSign | tpm2.FlagNoDA | tpm2.FlagDecrypt |
			tpm2.FlagSensitiveDataOrigin |
			tpm2.FlagUserWithAuth,
		ECCParameters: &tpm2.ECCParams{
			CurveID: tpm2.CurveNISTP256,
			Point:   tpm2.ECPoint{X: big.NewInt(0), Y: big.NewInt(0)},
		},
	}
	//Default Ek Template as per
	//https://trustedcomputinggroup.org/wp-content/uploads/Credential_Profile_EK_V2.0_R14_published.pdf
	defaultEkTemplate = tpm2.Public{
		Type:    tpm2.AlgRSA,
		NameAlg: tpm2.AlgSHA256,
		Attributes: tpm2.FlagFixedTPM | tpm2.FlagFixedParent | tpm2.FlagSensitiveDataOrigin |
			tpm2.FlagAdminWithPolicy | tpm2.FlagRestricted | tpm2.FlagDecrypt,
		AuthPolicy: []byte{
			0x83, 0x71, 0x97, 0x67, 0x44, 0x84,
			0xB3, 0xF8, 0x1A, 0x90, 0xCC, 0x8D,
			0x46, 0xA5, 0xD7, 0x24, 0xFD, 0x52,
			0xD7, 0x6E, 0x06, 0x52, 0x0B, 0x64,
			0xF2, 0xA1, 0xDA, 0x1B, 0x33, 0x14,
			0x69, 0xAA,
		},
		RSAParameters: &tpm2.RSAParams{
			Symmetric: &tpm2.SymScheme{
				Alg:     tpm2.AlgAES,
				KeyBits: 128,
				Mode:    tpm2.AlgCFB,
			},
			KeyBits:    2048,
			ModulusRaw: make([]byte, 256),
		},
	}
	//This is for ActivateCredentials() usage(Decrypt key)
	defaultSrkTemplate = tpm2.Public{
		Type:    tpm2.AlgRSA,
		NameAlg: tpm2.AlgSHA256,
		Attributes: tpm2.FlagFixedTPM | tpm2.FlagFixedParent |
			tpm2.FlagSensitiveDataOrigin | tpm2.FlagUserWithAuth |
			tpm2.FlagRestricted | tpm2.FlagDecrypt | tpm2.FlagNoDA,
		RSAParameters: &tpm2.RSAParams{
			Symmetric: &tpm2.SymScheme{
				Alg:     tpm2.AlgAES,
				KeyBits: 128,
				Mode:    tpm2.AlgCFB,
			},
			KeyBits:    2048,
			ModulusRaw: make([]byte, 256),
		},
	}
	//This is a restricted signing key, for PCR Quote and other such uses
	defaultAkTemplate = tpm2.Public{
		Type:    tpm2.AlgRSA,
		NameAlg: tpm2.AlgSHA256,
		Attributes: tpm2.FlagFixedTPM | tpm2.FlagFixedParent |
			tpm2.FlagSensitiveDataOrigin | tpm2.FlagUserWithAuth |
			tpm2.FlagRestricted | tpm2.FlagSign | tpm2.FlagNoDA,
		RSAParameters: &tpm2.RSAParams{
			Sign: &tpm2.SigScheme{
				Alg:  tpm2.AlgRSASSA,
				Hash: tpm2.AlgSHA256,
			},
			KeyBits:    2048,
			ModulusRaw: make([]byte, 256),
		},
	}
	debug         = false
	debugOverride bool // From command line arg
)

//Refer to https://trustedcomputinggroup.org/wp-content/uploads/TCG-TPM-Vendor-ID-Registry-Version-1.01-Revision-1.00.pdf
//These byte sequences in uint32 format is actually ASCII representation of TPM
//vendor ID. Since they are abbreviated names, we are having a map here to show
//a more verbose form of vendor name
var vendorRegistry = map[uint32]string{
	0x414D4400: "AMD",
	0x41544D4C: "Atmel",
	0x4252434D: "Broadcom",
	0x48504500: "HPE",
	0x49424d00: "IBM",
	0x49465800: "Infineon",
	0x494E5443: "Intel",
	0x4C454E00: "Lenovo",
	0x4D534654: "Microsoft",
	0x4E534D20: "National SC",
	0x4E545A00: "Nationz",
	0x4E544300: "Nuvoton",
	0x51434F4D: "Qualcomm",
	0x534D5343: "SMSC",
	0x53544D20: "ST Microelectronics",
	0x534D534E: "Samsung",
	0x534E5300: "Sinosun",
	0x54584E00: "Texas Instruments",
	0x57454300: "Winbond",
	0x524F4343: "Fuzhou Rockchip",
	0x474F4F47: "Google",
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

//IsTpmEnabled checks if TPM is being used by software for creating device cert
func IsTpmEnabled() bool {
	return fileExists(types.DeviceCertName) && !fileExists(types.DeviceKeyName)
}

//Helps creating various keys, according to the supplied template, and hierarchy
func createKey(keyHandle, ownerHandle tpmutil.Handle, template tpm2.Public, overwrite bool) error {
	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		log.Errorln(err)
		return err
	}
	defer rw.Close()
	if !overwrite {
		//don't overwrite if key already exists
		if _, _, _, err := tpm2.ReadPublic(rw, keyHandle); err == nil {
			return nil
		}
	}
	handle, _, err := tpm2.CreatePrimary(rw,
		tpm2.HandleOwner,
		pcrSelection,
		emptyPassword,
		emptyPassword,
		template)
	if err != nil {
		log.Errorf("create 0x%x failed: %s, do BIOS reset of TPM", keyHandle, err)
		return err
	}
	if err := tpm2.EvictControl(rw, emptyPassword,
		tpm2.HandleOwner,
		keyHandle,
		keyHandle); err != nil {
		log.Debugf("EvictControl failed: %v", err)
	}
	if err := tpm2.EvictControl(rw, emptyPassword,
		tpm2.HandleOwner, handle,
		keyHandle); err != nil {
		log.Errorf("EvictControl failed: %v, do BIOS reset of TPM", err)
		return err
	}
	return nil
}

func createDeviceKey() error {
	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		log.Errorln(err)
		return err
	}
	defer rw.Close()

	tpmOwnerPasswdBytes, err := ioutil.ReadFile(tpmCredentialsFileName)
	if err != nil {
		log.Fatalf("Reading from %s failed: %s", tpmCredentialsFileName, err)
		return err
	}

	tpmOwnerPasswd := string(tpmOwnerPasswdBytes)
	if len(tpmOwnerPasswd) > maxPasswdLength {
		tpmOwnerPasswd = tpmOwnerPasswd[0:maxPasswdLength]
	}

	//No previous key, create new one
	signerHandle, newPubKey, err := tpm2.CreatePrimary(rw,
		tpm2.HandleOwner,
		pcrSelection,
		emptyPassword,
		tpmOwnerPasswd,
		defaultKeyParams)
	if err != nil {
		log.Errorf("CreatePrimary failed: %s, do BIOS reset of TPM", err)
		return err
	}
	if err := tpm2.EvictControl(rw, emptyPassword,
		tpm2.HandleOwner,
		TpmDeviceKeyHdl,
		TpmDeviceKeyHdl); err != nil {
		log.Errorf("EvictControl failed: %v", err)
	}
	if err := tpm2.EvictControl(rw, emptyPassword,
		tpm2.HandleOwner, signerHandle,
		TpmDeviceKeyHdl); err != nil {
		log.Errorf("EvictControl failed: %v, do BIOS reset of TPM", err)
		return err
	}

	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(newPubKey)
	err = ioutil.WriteFile(TpmPubKeyName, pubKeyBytes, 0644)
	if err != nil {
		log.Errorf("Error in writing TPM public key to file: %v", err)
		return err
	}

	return nil
}

//TpmSign is used by external packages to get a digest signed by
//device key in TPM
func TpmSign(digest []byte) (*big.Int, *big.Int, error) {

	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return nil, nil, err
	}
	defer rw.Close()

	tpmOwnerPasswdBytes, err := ioutil.ReadFile(tpmCredentialsFileName)
	if err != nil {
		log.Fatalf("Reading from %s failed: %s", tpmCredentialsFileName, err)
		return nil, nil, err
	}
	tpmOwnerPasswd := string(tpmOwnerPasswdBytes)
	if len(tpmOwnerPasswd) > maxPasswdLength {
		tpmOwnerPasswd = tpmOwnerPasswd[0:maxPasswdLength]
	}

	//XXX This "32" should really come from Hash algo used.
	if len(digest) > 32 {
		digest = digest[:32]
	}

	scheme := &tpm2.SigScheme{
		Alg:  tpm2.AlgECDSA,
		Hash: tpm2.AlgSHA256,
	}
	sig, err := tpm2.Sign(rw, TpmDeviceKeyHdl,
		tpmOwnerPasswd, digest, scheme)
	if err != nil {
		log.Errorln("Sign using TPM failed")
		return nil, nil, err
	}
	return sig.ECC.R, sig.ECC.S, nil
}

func writeDeviceCert() error {

	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return err
	}
	defer rw.Close()

	if err := tpm2.NVUndefineSpace(rw, emptyPassword,
		tpm2.HandleOwner, TpmDeviceCertHdl,
	); err != nil {
		log.Debugf("NVUndefineSpace failed: %v", err)
	}

	deviceCertBytes, err := ioutil.ReadFile(TpmDeviceCertFileName)
	if err != nil {
		log.Errorf("Failed to read device cert file: %v", err)
		return err
	}

	// Define space in NV storage and clean up afterwards or subsequent runs will fail.
	if err := tpm2.NVDefineSpace(rw,
		tpm2.HandleOwner,
		TpmDeviceCertHdl,
		emptyPassword,
		emptyPassword,
		nil,
		tpm2.AttrOwnerWrite|tpm2.AttrOwnerRead,
		uint16(len(deviceCertBytes)),
	); err != nil {
		log.Errorf("NVDefineSpace failed: %v", err)
		return err
	}

	// Write the data
	if err := tpm2.NVWrite(rw, tpm2.HandleOwner, TpmDeviceCertHdl,
		emptyPassword, deviceCertBytes, 0); err != nil {
		log.Errorf("NVWrite failed: %v", err)
		return err
	}
	return nil
}

func readDeviceCert() error {

	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return err
	}
	defer rw.Close()

	// Read all of the data with NVReadEx
	deviceCertBytes, err := tpm2.NVReadEx(rw, TpmDeviceCertHdl,
		tpm2.HandleOwner, emptyPassword, 0)
	if err != nil {
		log.Errorf("NVReadEx failed: %v", err)
		return err
	}

	err = ioutil.WriteFile(TpmDeviceCertFileName, deviceCertBytes, 0644)
	if err != nil {
		log.Errorf("Writing to device cert file failed: %v", err)
		return err
	}

	return nil
}

func genCredentials() error {
	//First try to read from TPM, if it was stored earlier
	err := readCredentials()
	if err != nil {
		// Generate a new uuid
		out, err := exec.Command("uuidgen").Output()
		if err != nil {
			log.Fatalf("Error in generating uuid, %v", err)
			return err
		}
		//Write uuid to credentials file for faster access
		err = ioutil.WriteFile(tpmCredentialsFileName, out, 0644)
		if err != nil {
			log.Errorf("Writing to credentials file failed: %v", err)
			return err
		}
		//Write credentials to TPM for permenant storage.
		err = writeCredentials()
		if err != nil {
			log.Errorf("Writing credentials to TPM failed: %v", err)
			return err
		}
	}
	return nil
}

func writeCredentials() error {

	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return err
	}
	defer rw.Close()

	if err := tpm2.NVUndefineSpace(rw, emptyPassword,
		tpm2.HandleOwner, TpmPasswdHdl,
	); err != nil {
		log.Debugf("NVUndefineSpace failed: %v", err)
	}

	tpmCredentialBytes, err := ioutil.ReadFile(tpmCredentialsFileName)
	if err != nil {
		log.Errorf("Failed to read credentials file: %v", err)
		return err
	}

	// Define space in NV storage and clean up afterwards or subsequent runs will fail.
	if err := tpm2.NVDefineSpace(rw,
		tpm2.HandleOwner,
		TpmPasswdHdl,
		emptyPassword,
		emptyPassword,
		nil,
		tpm2.AttrOwnerWrite|tpm2.AttrOwnerRead,
		uint16(len(tpmCredentialBytes)),
	); err != nil {
		log.Errorf("NVDefineSpace failed: %v", err)
		return err
	}

	// Write the data
	if err := tpm2.NVWrite(rw, tpm2.HandleOwner, TpmPasswdHdl,
		emptyPassword, tpmCredentialBytes, 0); err != nil {
		log.Errorf("NVWrite failed: %v", err)
		return err
	}
	return nil
}

func readCredentials() error {

	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return err
	}
	defer rw.Close()

	// Read all of the data with NVReadEx
	tpmCredentialBytes, err := tpm2.NVReadEx(rw, TpmPasswdHdl,
		tpm2.HandleOwner, emptyPassword, 0)
	if err != nil {
		log.Errorf("NVReadEx failed: %v", err)
		return err
	}

	err = ioutil.WriteFile(tpmCredentialsFileName, tpmCredentialBytes, 0644)
	if err != nil {
		log.Errorf("Writing to credentials file failed: %v", err)
		return err
	}

	return nil
}

func getRandom(numBytes uint16) ([]byte, error) {
	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return nil, err
	}
	defer rw.Close()
	return tpm2.GetRandom(rw, numBytes)
}

//FetchVaultKey retreives TPM part of the vault key
func FetchVaultKey() ([]byte, error) {
	//First try to read from TPM, if it was stored earlier
	key, err := readDiskKey()
	if err != nil {
		key, err = getRandom(vaultKeyLength)
		if err != nil {
			log.Errorf("Error in generating random number: %v", err)
			return nil, err
		}
		err = writeDiskKey(key)
		if err != nil {
			log.Errorf("Writing Disk Key to TPM failed: %v", err)
			return nil, err
		}
	}
	return key, nil
}

func writeDiskKey(key []byte) error {
	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return err
	}
	defer rw.Close()

	if err := tpm2.NVUndefineSpace(rw, emptyPassword,
		tpm2.HandleOwner, TpmDiskKeyHdl,
	); err != nil {
		log.Debugf("NVUndefineSpace failed: %v", err)
	}

	// Define space in NV storage and clean up afterwards or subsequent runs will fail.
	if err := tpm2.NVDefineSpace(rw,
		tpm2.HandleOwner,
		TpmDiskKeyHdl,
		emptyPassword,
		emptyPassword,
		nil,
		tpm2.AttrOwnerWrite|tpm2.AttrOwnerRead,
		uint16(len(key)),
	); err != nil {
		log.Errorf("NVDefineSpace failed: %v", err)
		return err
	}

	// Write the data
	if err := tpm2.NVWrite(rw, tpm2.HandleOwner, TpmDiskKeyHdl,
		emptyPassword, key, 0); err != nil {
		log.Errorf("NVWrite failed: %v", err)
		return err
	}
	return nil
}

func readDiskKey() ([]byte, error) {
	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return nil, err
	}
	defer rw.Close()

	// Read all of the data with NVReadEx
	keyBytes, err := tpm2.NVReadEx(rw, TpmDiskKeyHdl,
		tpm2.HandleOwner, emptyPassword, 0)
	if err != nil {
		log.Errorf("NVReadEx failed: %v", err)
		return nil, err
	}
	return keyBytes, nil
}

//till we have next version of go-tpm released, use this
const (
	tpmPropertyManufacturer tpm2.TPMProp = 0x105
	tpmPropertyVendorStr1   tpm2.TPMProp = 0x106
	tpmPropertyVendorStr2   tpm2.TPMProp = 0x107
	tpmPropertyFirmVer1     tpm2.TPMProp = 0x10b
	tpmPropertyFirmVer2     tpm2.TPMProp = 0x10c
)

func getModelName(vendorValue1 uint32, vendorValue2 uint32) string {
	uintToByteArr := func(value uint32) []byte {
		get8 := func(val uint32, offset uint32) uint8 {
			return (uint8)((val >> ((3 - offset) * 8)) & 0xff)
		}
		var i uint32
		var bytes []byte
		for i = 0; i < uint32(unsafe.Sizeof(value)); i++ {
			c := get8(value, i)
			bytes = append(bytes, c)
		}
		return bytes
	}
	var model []byte
	model = append(model, uintToByteArr(vendorValue1)...)
	model = append(model, uintToByteArr(vendorValue2)...)
	return string(model)
}

func getFirmwareVersion(v1 uint32, v2 uint32) string {
	get16 := func(val uint32, offset uint32) uint16 {
		return uint16((val >> ((1 - offset) * 16)) & 0xFFFF)
	}
	return fmt.Sprintf("%d.%d.%d.%d", get16(v1, 0), get16(v1, 1),
		get16(v2, 0), get16(v2, 1))
}

func getTpmProperty(propID tpm2.TPMProp) (uint32, error) {

	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return 0, err
	}
	defer rw.Close()

	v, _, err := tpm2.GetCapability(rw, tpm2.CapabilityTPMProperties,
		1, uint32(propID))
	if err != nil {
		return 0, err
	}
	prop, ok := v[0].(tpm2.TaggedProperty)
	if !ok {
		return 0, fmt.Errorf("Unable to fetch property %d", propID)
	}
	return prop.Value, nil
}

//FetchTpmSwStatus returns states reflecting SW usage of TPM
func FetchTpmSwStatus() info.HwSecurityModuleStatus {
	_, err := os.Stat(TpmDevicePath)
	if err != nil {
		//No TPM found on this system
		return info.HwSecurityModuleStatus_NOTFOUND
	}
	if IsTpmEnabled() {
		//TPM is found and is used by software
		return info.HwSecurityModuleStatus_ENABLED
	}

	//TPM is found but not being used by software
	return info.HwSecurityModuleStatus_DISABLED
}

//FetchTpmHwInfo returns TPM Hardware properties in a string
func FetchTpmHwInfo() (string, error) {

	//If we had done this earlier, return the last result
	if tpmHwInfo != "" {
		return tpmHwInfo, nil
	}

	//Take care of non-TPM platforms
	_, err := os.Stat(TpmDevicePath)
	if err != nil {
		tpmHwInfo = "Not Available"
		return tpmHwInfo, nil
	}

	//First time. Fetch it from TPM and cache it.
	v1, err := getTpmProperty(tpmPropertyManufacturer)
	if err != nil {
		return "", err
	}
	v2, err := getTpmProperty(tpmPropertyVendorStr1)
	if err != nil {
		return "", err
	}
	v3, err := getTpmProperty(tpmPropertyVendorStr2)
	if err != nil {
		return "", err
	}
	v4, err := getTpmProperty(tpmPropertyFirmVer1)
	if err != nil {
		return "", err
	}
	v5, err := getTpmProperty(tpmPropertyFirmVer2)
	if err != nil {
		return "", err
	}
	tpmHwInfo = fmt.Sprintf("%s-%s, FW Version %s", vendorRegistry[v1],
		getModelName(v2, v3),
		getFirmwareVersion(v4, v5))

	return tpmHwInfo, nil
}

func printNVProperties() {
	if nvMaxSize, err := getTpmProperty(tpm2.NVMaxBufferSize); err != nil {
		fmt.Printf("NV Max Size: 0x%X\n", nvMaxSize)
	}
	if nvIdxFirst, err := getTpmProperty(tpm2.NVIndexFirst); err != nil {
		fmt.Printf("NV Index First: 0x%X\n", nvIdxFirst)
	}
	if nvIdxLast, err := getTpmProperty(tpm2.NVIndexLast); err != nil {
		fmt.Printf("NV Index Last: 0x%X\n", nvIdxLast)
	}
}

func printCapability() {
	hwInfoStr, err := FetchTpmHwInfo()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(hwInfoStr)
	}
	//XXX Not working, commenting for now
	//printNVProperties()
}

func printPCRs() {
	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		return
	}
	defer rw.Close()
	for i := 0; i < 23; i++ {
		pcrVal, err := tpm2.ReadPCR(rw, i, tpm2.AlgSHA256)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("PCR %d: Value: 0x%X\n", i, pcrVal)
	}
	attestData, pcrQuote, err := tpm2.Quote(rw, TpmAKHdl,
		emptyPassword,
		emptyPassword,
		[]byte("nonce"),
		pcrListForQuote,
		tpm2.AlgNull)
	if err != nil {
		fmt.Printf("Error in creating quote: %v\n", err)
	} else {
		fmt.Printf("attestData = %v\n", attestData)
		fmt.Printf("pcrQuote = %v, %v\n", pcrQuote.Alg, *pcrQuote.RSA)
	}
}

func testTpmEcdhSupport() error {
	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		log.Errorln(err)
		return err
	}
	defer rw.Close()

	z, p, err := tpm2.GenerateSharedECCSecret(rw, TpmDeviceKeyHdl, emptyPassword)
	if err != nil {
		fmt.Printf("generating Shared Secret failed: %s", err)
		return err
	}
	tpmOwnerPasswdBytes, err := ioutil.ReadFile(tpmCredentialsFileName)
	if err != nil {
		fmt.Printf("Reading from %s failed: %s", tpmCredentialsFileName, err)
		return err
	}

	tpmOwnerPasswd := string(tpmOwnerPasswdBytes)
	if len(tpmOwnerPasswd) > maxPasswdLength {
		tpmOwnerPasswd = tpmOwnerPasswd[0:maxPasswdLength]
	}

	z1, err := tpm2.RecoverSharedECCSecret(rw, TpmDeviceKeyHdl, tpmOwnerPasswd, p)
	if err != nil {
		fmt.Printf("recovering Shared Secret failed: %s", err)
		return err
	}
	fmt.Println(reflect.DeepEqual(z, z1))
	return nil
}

func aesEncrypt(ciphertext, plaintext, key, iv []byte) error {
	aesBlockEncrypter, err := aes.NewCipher([]byte(key))
	if err != nil {
		return err
	}
	aesEncrypter := cipher.NewCFBEncrypter(aesBlockEncrypter, iv)
	aesEncrypter.XORKeyStream(ciphertext, plaintext)
	return nil
}

func aesDecrypt(plaintext, ciphertext, key, iv []byte) error {
	aesBlockDecrypter, err := aes.NewCipher([]byte(key))
	if err != nil {
		return err
	}
	aesDecrypter := cipher.NewCFBDecrypter(aesBlockDecrypter, iv)
	aesDecrypter.XORKeyStream(plaintext, ciphertext)
	return nil
}

func sha256FromECPoint(X, Y *big.Int) [32]byte {
	var bytes = make([]byte, 0)
	bytes = append(bytes, X.Bytes()...)
	bytes = append(bytes, Y.Bytes()...)
	return sha256.Sum256(bytes)
}

//DecryptSecretWithEcdhKey recovers plaintext from given X, Y, iv and the ciphertext
func DecryptSecretWithEcdhKey(X, Y *big.Int, iv, ciphertext, plaintext []byte) error {
	rw, err := tpm2.OpenTPM(TpmDevicePath)
	if err != nil {
		log.Errorln(err)
		return err
	}
	defer rw.Close()

	p := tpm2.ECPoint{X: X, Y: Y}
	tpmOwnerPasswdBytes, err := ioutil.ReadFile(tpmCredentialsFileName)
	if err != nil {
		fmt.Printf("Reading from %s failed: %s", tpmCredentialsFileName, err)
		return err
	}
	//Recover the key, and decrypt the message (EVE node Part)
	tpmOwnerPasswd := string(tpmOwnerPasswdBytes)
	if len(tpmOwnerPasswd) > maxPasswdLength {
		tpmOwnerPasswd = tpmOwnerPasswd[0:maxPasswdLength]
	}
	z, err := tpm2.RecoverSharedECCSecret(rw, TpmDeviceKeyHdl, tpmOwnerPasswd, p)
	if err != nil {
		fmt.Printf("recovering Shared Secret failed: %s", err)
		return err
	}
	decryptKey := sha256FromECPoint(z.X, z.Y)
	return aesDecrypt(plaintext, ciphertext, decryptKey[:], iv)
}

//Test ECDH key exchange and a symmetric cipher based on ECDH
func testEcdhAES() error {
	//Simulate Controller generating an ephemeral key
	privateA, publicAX, publicAY, err := elliptic.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Printf("Failed to generate A private/public key pair: %s\n", err)
	}

	//read public key from device certificate
	certBytes, err := ioutil.ReadFile("/config/device.cert.pem")
	if err != nil {
		fmt.Printf("error in reading device cert file: %v", err)
		return err
	}
	block, _ := pem.Decode(certBytes)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Printf("error in parsing device cert file: %v", err)
		return err
	}
	publicB := cert.PublicKey.(*ecdsa.PublicKey)

	//multiply privateA with publicB (Controller Part)
	X, Y := elliptic.P256().Params().ScalarMult(publicB.X, publicB.Y, privateA)

	fmt.Printf("publicAX, publicAY, X/Y = %v, %v, %v, %v\n", publicAX, publicAY, X, Y)
	encryptKey := sha256FromECPoint(X, Y)
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		fmt.Printf("Unable to generate Initial Value %v\n", err)
	}

	msg := []byte("this is the secret")
	ciphertext := make([]byte, len(msg))
	aesEncrypt(ciphertext, msg, encryptKey[:], iv)

	recoveredMsg := make([]byte, len(ciphertext))
	err = DecryptSecretWithEcdhKey(publicAX, publicAY, iv, ciphertext, recoveredMsg)
	if err != nil {
		fmt.Printf("Decryption failed with error %v\n", err)
		return err
	}
	fmt.Println(reflect.DeepEqual(msg, recoveredMsg))
	return nil
}

func Run(ps *pubsub.PubSub) {
	curpartPtr := flag.String("c", "", "Current partition")
	debugPtr := flag.Bool("d", false, "Debug flag")
	flag.Parse()
	debug = *debugPtr
	debugOverride = debug
	if debugOverride {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	curpart := *curpartPtr
	// Sending json log format to stdout
	logf, err := agentlog.Init("tpmmgr", curpart)
	if err != nil {
		log.Fatal(err)
	}
	defer logf.Close()

	if len(flag.Args()) == 0 {
		log.Fatal("Insufficient arguments")
	}
	switch flag.Args()[0] {
	case "genKey":
		if err = createDeviceKey(); err != nil {
			log.Fatalf("Error in creating device primary key: %v ", err)
		}
		if err = createKey(TpmEKHdl, tpm2.HandleEndorsement, defaultEkTemplate, true); err != nil {
			log.Fatalf("Error in creating Endorsement key: %v ", err)
		}
		if err = createKey(TpmSRKHdl, tpm2.HandleOwner, defaultSrkTemplate, true); err != nil {
			log.Fatalf("Error in creating Srk key: %v ", err)
		}
		if err = createKey(TpmAKHdl, tpm2.HandleOwner, defaultAkTemplate, true); err != nil {
			log.Fatalf("Error in creating Attestation key: %v ", err)
		}
	case "readDeviceCert":
		if err = readDeviceCert(); err != nil {
			log.Fatalf("Error in reading device cert: %v", err)
		}
	case "writeDeviceCert":
		if err = writeDeviceCert(); err != nil {
			log.Fatalf("Error in writing device cert: %v", err)
		}
	case "readCredentials":
		if err = readCredentials(); err != nil {
			log.Fatalf("Error in reading credentials: %v", err)
		}
	case "genCredentials":
		if err = genCredentials(); err != nil {
			log.Fatalf("Error in generating credentials: %v", err)
		}
	case "runAsService":
		log.Infof("Starting %s\n", agentName)

		if err := pidfile.CheckAndCreatePidfile(agentName); err != nil {
			log.Fatal(err)
		}

		// Run a periodic timer so we always update StillRunning
		stillRunning := time.NewTicker(15 * time.Second)
		agentlog.StillRunning(agentName, warningTime, errorTime)

		// Context to pass around
		ctx := tpmMgrContext{}

		// Look for global config such as log levels
		subGlobalConfig, err := pubsublegacy.Subscribe("", types.GlobalConfig{},
			false, &ctx, &pubsub.SubscriptionOptions{
				CreateHandler: handleGlobalConfigModify,
				ModifyHandler: handleGlobalConfigModify,
				DeleteHandler: handleGlobalConfigDelete,
				WarningTime:   warningTime,
				ErrorTime:     errorTime,
			})
		if err != nil {
			log.Fatal(err)
		}
		ctx.subGlobalConfig = subGlobalConfig
		subGlobalConfig.Activate()

		// Pick up debug aka log level before we start real work
		for !ctx.GCInitialized {
			log.Infof("waiting for GCInitialized")
			select {
			case change := <-subGlobalConfig.MsgChan():
				subGlobalConfig.ProcessChange(change)
			case <-stillRunning.C:
			}
			agentlog.StillRunning(agentName, warningTime, errorTime)
		}
		log.Infof("processed GlobalConfig")

		if err = createKey(TpmEKHdl, tpm2.HandleEndorsement, defaultEkTemplate, false); err != nil {
			log.Fatalf("Error in creating Endorsement key: %v ", err)
		}
		if err = createKey(TpmSRKHdl, tpm2.HandleOwner, defaultSrkTemplate, false); err != nil {
			log.Fatalf("Error in creating Srk key: %v ", err)
		}
		if err = createKey(TpmAKHdl, tpm2.HandleOwner, defaultAkTemplate, false); err != nil {
			log.Fatalf("Error in creating Attestation key: %v ", err)
		}
		for {
			select {
			case <-stillRunning.C:
				agentlog.StillRunning(agentName, warningTime, errorTime)
			}
		}
	case "printCapability":
		printCapability()
	case "printPCRs":
		printPCRs()
	case "testTpmEcdhSupport":
		testTpmEcdhSupport()
	case "testEcdhAES":
		testEcdhAES()
	default:
		log.Fatalf("Unknown argument %s", flag.Args()[0])
	}
}

// Handles both create and modify events
func handleGlobalConfigModify(ctxArg interface{}, key string,
	statusArg interface{}) {

	ctx := ctxArg.(*tpmMgrContext)
	if key != "global" {
		log.Infof("handleGlobalConfigModify: ignoring %s\n", key)
		return
	}
	log.Infof("handleGlobalConfigModify for %s\n", key)
	var gcp *types.GlobalConfig
	debug, gcp = agentlog.HandleGlobalConfig(ctx.subGlobalConfig, agentName,
		debugOverride)
	if gcp != nil {
		ctx.GCInitialized = true
	}
	log.Infof("handleGlobalConfigModify done for %s\n", key)
}

func handleGlobalConfigDelete(ctxArg interface{}, key string,
	statusArg interface{}) {

	ctx := ctxArg.(*tpmMgrContext)
	if key != "global" {
		log.Infof("handleGlobalConfigDelete: ignoring %s\n", key)
		return
	}
	log.Infof("handleGlobalConfigDelete for %s\n", key)
	debug, _ = agentlog.HandleGlobalConfig(ctx.subGlobalConfig, agentName,
		debugOverride)
	log.Infof("handleGlobalConfigDelete done for %s\n", key)
}
