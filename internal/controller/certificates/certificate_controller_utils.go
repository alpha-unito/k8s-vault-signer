package certificates

import (
	capi "k8s.io/api/certificates/v1"
	v1 "k8s.io/api/core/v1"
)

func IsCertificateRequestApproved(csr *capi.CertificateSigningRequest) bool {
	approved, denied := GetCertApprovalCondition(&csr.Status)
	return approved && !denied
}

func HasTrueCondition(csr *capi.CertificateSigningRequest, conditionType capi.RequestConditionType) bool {
	for _, c := range csr.Status.Conditions {
		if c.Type == conditionType && (len(c.Status) == 0 || c.Status == v1.ConditionTrue) {
			return true
		}
	}
	return false
}

func GetCertApprovalCondition(status *capi.CertificateSigningRequestStatus) (approved bool, denied bool) {
	for _, c := range status.Conditions {
		if c.Type == capi.CertificateApproved {
			approved = true
		}
		if c.Type == capi.CertificateDenied {
			denied = true
		}
	}
	return
}
