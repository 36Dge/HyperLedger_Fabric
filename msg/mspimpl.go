package msg

import (
	"crypto/x509"
	"crypto/x509/pkix"
)

// mspSetupFuncType is the prototype of the setup function
type mspSetupFuncType func(config *m.FabricMSPConfig) error

// validateIdentityOUsFuncType is the prototype of the function to validate identity's OUs
type validateIdentityOUsFuncType func(id *identity) error

// satisfiesPrincipalInternalFuncType is the prototype of the function to check if principals are satisfied
type satisfiesPrincipalInternalFuncType func(id Identity, principal *m.MSPPrincipal) error

//setupAdminInternalFuncType is a prototype of the function to setup the admins
type setupAdminInternalFuncType func(conf *m.FabricMSPConfig) error


// This is an instantiation of an MSP that
// uses BCCSP for its cryptographic primitives.
type bccspmsp struct {
	// version specifies the behaviour of this msp
	version MSPVersion
	// The following function pointers are used to change the behaviour
	// of this MSP depending on its version.
	// internalSetupFunc is the pointer to the setup function
	internalSetupFunc mspSetupFuncType

	// internalValidateIdentityOusFunc is the pointer to the function to validate identity's OUs
	internalValidateIdentityOusFunc validateIdentityOUsFuncType

	// internalSatisfiesPrincipalInternalFunc is the pointer to the function to check if principals are satisfied
	internalSatisfiesPrincipalInternalFunc satisfiesPrincipalInternalFuncType

	// internalSetupAdmin is the pointer to the function that setup the administrators of this msp
	internalSetupAdmin setupAdminInternalFuncType

	// list of CA certs we trust
	rootCerts []Identity

	// list of intermediate certs we trust
	intermediateCerts []Identity

	// list of CA TLS certs we trust
	tlsRootCerts [][]byte

	// list of intermediate TLS certs we trust
	tlsIntermediateCerts [][]byte

	// certificationTreeInternalNodesMap whose keys correspond to the raw material
	// (DER representation) of a certificate casted to a string, and whose values
	// are boolean. True means that the certificate is an internal node of the certification tree.
	// False means that the certificate corresponds to a leaf of the certification tree.
	certificationTreeInternalNodesMap map[string]bool

	// list of signing identities
	signer SigningIdentity

	// list of admin identities
	admins []Identity

	// the crypto provider
	bccsp bccsp.BCCSP

	// the provider identifier for this MSP
	name string

	// verification options for MSP members
	opts *x509.VerifyOptions

	// list of certificate revocation lists
	CRL []*pkix.CertificateList

	// list of OUs
	ouIdentifiers map[string][][]byte

	// cryptoConfig contains
	cryptoConfig *m.FabricCryptoConfig

	// NodeOUs configuration
	ouEnforcement bool
	// These are the OUIdentifiers of the clients, peers, admins and orderers.
	// They are used to tell apart these entities
	clientOU, peerOU, adminOU, ordererOU *OUIdentifier
}

