// This file is generated by "./lib/proto/generate"

package proto

/*

Preload

*/

// PreloadRuleSetID Unique id.
type PreloadRuleSetID string

// PreloadRuleSet Corresponds to SpeculationRuleSet.
type PreloadRuleSet struct {
	// ID ...
	ID PreloadRuleSetID `json:"id"`

	// LoaderID Identifies a document which the rule set is associated with.
	LoaderID NetworkLoaderID `json:"loaderId"`

	// SourceText Source text of JSON representing the rule set. If it comes from
	// `<script>` tag, it is the textContent of the node. Note that it is
	// a JSON for valid case.
	//
	// See also:
	// - https://wicg.github.io/nav-speculation/speculation-rules.html
	// - https://github.com/WICG/nav-speculation/blob/main/triggers.md
	SourceText string `json:"sourceText"`

	// BackendNodeID (optional) A speculation rule set is either added through an inline
	// `<script>` tag or through an external resource via the
	// 'Speculation-Rules' HTTP header. For the first case, we include
	// the BackendNodeId of the relevant `<script>` tag. For the second
	// case, we include the external URL where the rule set was loaded
	// from, and also RequestId if Network domain is enabled.
	//
	// See also:
	// - https://wicg.github.io/nav-speculation/speculation-rules.html#speculation-rules-script
	// - https://wicg.github.io/nav-speculation/speculation-rules.html#speculation-rules-header
	BackendNodeID DOMBackendNodeID `json:"backendNodeId,omitempty"`

	// URL (optional) ...
	URL string `json:"url,omitempty"`

	// RequestID (optional) ...
	RequestID NetworkRequestID `json:"requestId,omitempty"`

	// ErrorType (optional) Error information
	// `errorMessage` is null iff `errorType` is null.
	ErrorType PreloadRuleSetErrorType `json:"errorType,omitempty"`

	// ErrorMessage (deprecated) (optional) TODO(https://crbug.com/1425354): Replace this property with structured error.
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// PreloadRuleSetErrorType ...
type PreloadRuleSetErrorType string

const (
	// PreloadRuleSetErrorTypeSourceIsNotJSONObject enum const.
	PreloadRuleSetErrorTypeSourceIsNotJSONObject PreloadRuleSetErrorType = "SourceIsNotJsonObject"

	// PreloadRuleSetErrorTypeInvalidRulesSkipped enum const.
	PreloadRuleSetErrorTypeInvalidRulesSkipped PreloadRuleSetErrorType = "InvalidRulesSkipped"
)

// PreloadSpeculationAction The type of preloading attempted. It corresponds to
// mojom::SpeculationAction (although PrefetchWithSubresources is omitted as it
// isn't being used by clients).
type PreloadSpeculationAction string

const (
	// PreloadSpeculationActionPrefetch enum const.
	PreloadSpeculationActionPrefetch PreloadSpeculationAction = "Prefetch"

	// PreloadSpeculationActionPrerender enum const.
	PreloadSpeculationActionPrerender PreloadSpeculationAction = "Prerender"
)

// PreloadSpeculationTargetHint Corresponds to mojom::SpeculationTargetHint.
// See https://github.com/WICG/nav-speculation/blob/main/triggers.md#window-name-targeting-hints
type PreloadSpeculationTargetHint string

const (
	// PreloadSpeculationTargetHintBlank enum const.
	PreloadSpeculationTargetHintBlank PreloadSpeculationTargetHint = "Blank"

	// PreloadSpeculationTargetHintSelf enum const.
	PreloadSpeculationTargetHintSelf PreloadSpeculationTargetHint = "Self"
)

// PreloadPreloadingAttemptKey A key that identifies a preloading attempt.
//
// The url used is the url specified by the trigger (i.e. the initial URL), and
// not the final url that is navigated to. For example, prerendering allows
// same-origin main frame navigations during the attempt, but the attempt is
// still keyed with the initial URL.
type PreloadPreloadingAttemptKey struct {
	// LoaderID ...
	LoaderID NetworkLoaderID `json:"loaderId"`

	// Action ...
	Action PreloadSpeculationAction `json:"action"`

	// URL ...
	URL string `json:"url"`

	// TargetHint (optional) ...
	TargetHint PreloadSpeculationTargetHint `json:"targetHint,omitempty"`
}

// PreloadPreloadingAttemptSource Lists sources for a preloading attempt, specifically the ids of rule sets
// that had a speculation rule that triggered the attempt, and the
// BackendNodeIds of <a href> or <area href> elements that triggered the
// attempt (in the case of attempts triggered by a document rule). It is
// possible for multiple rule sets and links to trigger a single attempt.
type PreloadPreloadingAttemptSource struct {
	// Key ...
	Key *PreloadPreloadingAttemptKey `json:"key"`

	// RuleSetIDs ...
	RuleSetIDs []PreloadRuleSetID `json:"ruleSetIds"`

	// NodeIDs ...
	NodeIDs []DOMBackendNodeID `json:"nodeIds"`
}

// PreloadPrerenderFinalStatus List of FinalStatus reasons for Prerender2.
type PreloadPrerenderFinalStatus string

const (
	// PreloadPrerenderFinalStatusActivated enum const.
	PreloadPrerenderFinalStatusActivated PreloadPrerenderFinalStatus = "Activated"

	// PreloadPrerenderFinalStatusDestroyed enum const.
	PreloadPrerenderFinalStatusDestroyed PreloadPrerenderFinalStatus = "Destroyed"

	// PreloadPrerenderFinalStatusLowEndDevice enum const.
	PreloadPrerenderFinalStatusLowEndDevice PreloadPrerenderFinalStatus = "LowEndDevice"

	// PreloadPrerenderFinalStatusInvalidSchemeRedirect enum const.
	PreloadPrerenderFinalStatusInvalidSchemeRedirect PreloadPrerenderFinalStatus = "InvalidSchemeRedirect"

	// PreloadPrerenderFinalStatusInvalidSchemeNavigation enum const.
	PreloadPrerenderFinalStatusInvalidSchemeNavigation PreloadPrerenderFinalStatus = "InvalidSchemeNavigation"

	// PreloadPrerenderFinalStatusNavigationRequestBlockedByCsp enum const.
	PreloadPrerenderFinalStatusNavigationRequestBlockedByCsp PreloadPrerenderFinalStatus = "NavigationRequestBlockedByCsp"

	// PreloadPrerenderFinalStatusMainFrameNavigation enum const.
	PreloadPrerenderFinalStatusMainFrameNavigation PreloadPrerenderFinalStatus = "MainFrameNavigation"

	// PreloadPrerenderFinalStatusMojoBinderPolicy enum const.
	PreloadPrerenderFinalStatusMojoBinderPolicy PreloadPrerenderFinalStatus = "MojoBinderPolicy"

	// PreloadPrerenderFinalStatusRendererProcessCrashed enum const.
	PreloadPrerenderFinalStatusRendererProcessCrashed PreloadPrerenderFinalStatus = "RendererProcessCrashed"

	// PreloadPrerenderFinalStatusRendererProcessKilled enum const.
	PreloadPrerenderFinalStatusRendererProcessKilled PreloadPrerenderFinalStatus = "RendererProcessKilled"

	// PreloadPrerenderFinalStatusDownload enum const.
	PreloadPrerenderFinalStatusDownload PreloadPrerenderFinalStatus = "Download"

	// PreloadPrerenderFinalStatusTriggerDestroyed enum const.
	PreloadPrerenderFinalStatusTriggerDestroyed PreloadPrerenderFinalStatus = "TriggerDestroyed"

	// PreloadPrerenderFinalStatusNavigationNotCommitted enum const.
	PreloadPrerenderFinalStatusNavigationNotCommitted PreloadPrerenderFinalStatus = "NavigationNotCommitted"

	// PreloadPrerenderFinalStatusNavigationBadHTTPStatus enum const.
	PreloadPrerenderFinalStatusNavigationBadHTTPStatus PreloadPrerenderFinalStatus = "NavigationBadHttpStatus"

	// PreloadPrerenderFinalStatusClientCertRequested enum const.
	PreloadPrerenderFinalStatusClientCertRequested PreloadPrerenderFinalStatus = "ClientCertRequested"

	// PreloadPrerenderFinalStatusNavigationRequestNetworkError enum const.
	PreloadPrerenderFinalStatusNavigationRequestNetworkError PreloadPrerenderFinalStatus = "NavigationRequestNetworkError"

	// PreloadPrerenderFinalStatusCancelAllHostsForTesting enum const.
	PreloadPrerenderFinalStatusCancelAllHostsForTesting PreloadPrerenderFinalStatus = "CancelAllHostsForTesting"

	// PreloadPrerenderFinalStatusDidFailLoad enum const.
	PreloadPrerenderFinalStatusDidFailLoad PreloadPrerenderFinalStatus = "DidFailLoad"

	// PreloadPrerenderFinalStatusStop enum const.
	PreloadPrerenderFinalStatusStop PreloadPrerenderFinalStatus = "Stop"

	// PreloadPrerenderFinalStatusSslCertificateError enum const.
	PreloadPrerenderFinalStatusSslCertificateError PreloadPrerenderFinalStatus = "SslCertificateError"

	// PreloadPrerenderFinalStatusLoginAuthRequested enum const.
	PreloadPrerenderFinalStatusLoginAuthRequested PreloadPrerenderFinalStatus = "LoginAuthRequested"

	// PreloadPrerenderFinalStatusUaChangeRequiresReload enum const.
	PreloadPrerenderFinalStatusUaChangeRequiresReload PreloadPrerenderFinalStatus = "UaChangeRequiresReload"

	// PreloadPrerenderFinalStatusBlockedByClient enum const.
	PreloadPrerenderFinalStatusBlockedByClient PreloadPrerenderFinalStatus = "BlockedByClient"

	// PreloadPrerenderFinalStatusAudioOutputDeviceRequested enum const.
	PreloadPrerenderFinalStatusAudioOutputDeviceRequested PreloadPrerenderFinalStatus = "AudioOutputDeviceRequested"

	// PreloadPrerenderFinalStatusMixedContent enum const.
	PreloadPrerenderFinalStatusMixedContent PreloadPrerenderFinalStatus = "MixedContent"

	// PreloadPrerenderFinalStatusTriggerBackgrounded enum const.
	PreloadPrerenderFinalStatusTriggerBackgrounded PreloadPrerenderFinalStatus = "TriggerBackgrounded"

	// PreloadPrerenderFinalStatusMemoryLimitExceeded enum const.
	PreloadPrerenderFinalStatusMemoryLimitExceeded PreloadPrerenderFinalStatus = "MemoryLimitExceeded"

	// PreloadPrerenderFinalStatusDataSaverEnabled enum const.
	PreloadPrerenderFinalStatusDataSaverEnabled PreloadPrerenderFinalStatus = "DataSaverEnabled"

	// PreloadPrerenderFinalStatusTriggerURLHasEffectiveURL enum const.
	PreloadPrerenderFinalStatusTriggerURLHasEffectiveURL PreloadPrerenderFinalStatus = "TriggerUrlHasEffectiveUrl"

	// PreloadPrerenderFinalStatusActivatedBeforeStarted enum const.
	PreloadPrerenderFinalStatusActivatedBeforeStarted PreloadPrerenderFinalStatus = "ActivatedBeforeStarted"

	// PreloadPrerenderFinalStatusInactivePageRestriction enum const.
	PreloadPrerenderFinalStatusInactivePageRestriction PreloadPrerenderFinalStatus = "InactivePageRestriction"

	// PreloadPrerenderFinalStatusStartFailed enum const.
	PreloadPrerenderFinalStatusStartFailed PreloadPrerenderFinalStatus = "StartFailed"

	// PreloadPrerenderFinalStatusTimeoutBackgrounded enum const.
	PreloadPrerenderFinalStatusTimeoutBackgrounded PreloadPrerenderFinalStatus = "TimeoutBackgrounded"

	// PreloadPrerenderFinalStatusCrossSiteRedirectInInitialNavigation enum const.
	PreloadPrerenderFinalStatusCrossSiteRedirectInInitialNavigation PreloadPrerenderFinalStatus = "CrossSiteRedirectInInitialNavigation"

	// PreloadPrerenderFinalStatusCrossSiteNavigationInInitialNavigation enum const.
	PreloadPrerenderFinalStatusCrossSiteNavigationInInitialNavigation PreloadPrerenderFinalStatus = "CrossSiteNavigationInInitialNavigation"

	// PreloadPrerenderFinalStatusSameSiteCrossOriginRedirectNotOptInInInitialNavigation enum const.
	PreloadPrerenderFinalStatusSameSiteCrossOriginRedirectNotOptInInInitialNavigation PreloadPrerenderFinalStatus = "SameSiteCrossOriginRedirectNotOptInInInitialNavigation"

	// PreloadPrerenderFinalStatusSameSiteCrossOriginNavigationNotOptInInInitialNavigation enum const.
	PreloadPrerenderFinalStatusSameSiteCrossOriginNavigationNotOptInInInitialNavigation PreloadPrerenderFinalStatus = "SameSiteCrossOriginNavigationNotOptInInInitialNavigation"

	// PreloadPrerenderFinalStatusActivationNavigationParameterMismatch enum const.
	PreloadPrerenderFinalStatusActivationNavigationParameterMismatch PreloadPrerenderFinalStatus = "ActivationNavigationParameterMismatch"

	// PreloadPrerenderFinalStatusActivatedInBackground enum const.
	PreloadPrerenderFinalStatusActivatedInBackground PreloadPrerenderFinalStatus = "ActivatedInBackground"

	// PreloadPrerenderFinalStatusEmbedderHostDisallowed enum const.
	PreloadPrerenderFinalStatusEmbedderHostDisallowed PreloadPrerenderFinalStatus = "EmbedderHostDisallowed"

	// PreloadPrerenderFinalStatusActivationNavigationDestroyedBeforeSuccess enum const.
	PreloadPrerenderFinalStatusActivationNavigationDestroyedBeforeSuccess PreloadPrerenderFinalStatus = "ActivationNavigationDestroyedBeforeSuccess"

	// PreloadPrerenderFinalStatusTabClosedByUserGesture enum const.
	PreloadPrerenderFinalStatusTabClosedByUserGesture PreloadPrerenderFinalStatus = "TabClosedByUserGesture"

	// PreloadPrerenderFinalStatusTabClosedWithoutUserGesture enum const.
	PreloadPrerenderFinalStatusTabClosedWithoutUserGesture PreloadPrerenderFinalStatus = "TabClosedWithoutUserGesture"

	// PreloadPrerenderFinalStatusPrimaryMainFrameRendererProcessCrashed enum const.
	PreloadPrerenderFinalStatusPrimaryMainFrameRendererProcessCrashed PreloadPrerenderFinalStatus = "PrimaryMainFrameRendererProcessCrashed"

	// PreloadPrerenderFinalStatusPrimaryMainFrameRendererProcessKilled enum const.
	PreloadPrerenderFinalStatusPrimaryMainFrameRendererProcessKilled PreloadPrerenderFinalStatus = "PrimaryMainFrameRendererProcessKilled"

	// PreloadPrerenderFinalStatusActivationFramePolicyNotCompatible enum const.
	PreloadPrerenderFinalStatusActivationFramePolicyNotCompatible PreloadPrerenderFinalStatus = "ActivationFramePolicyNotCompatible"

	// PreloadPrerenderFinalStatusPreloadingDisabled enum const.
	PreloadPrerenderFinalStatusPreloadingDisabled PreloadPrerenderFinalStatus = "PreloadingDisabled"

	// PreloadPrerenderFinalStatusBatterySaverEnabled enum const.
	PreloadPrerenderFinalStatusBatterySaverEnabled PreloadPrerenderFinalStatus = "BatterySaverEnabled"

	// PreloadPrerenderFinalStatusActivatedDuringMainFrameNavigation enum const.
	PreloadPrerenderFinalStatusActivatedDuringMainFrameNavigation PreloadPrerenderFinalStatus = "ActivatedDuringMainFrameNavigation"

	// PreloadPrerenderFinalStatusPreloadingUnsupportedByWebContents enum const.
	PreloadPrerenderFinalStatusPreloadingUnsupportedByWebContents PreloadPrerenderFinalStatus = "PreloadingUnsupportedByWebContents"

	// PreloadPrerenderFinalStatusCrossSiteRedirectInMainFrameNavigation enum const.
	PreloadPrerenderFinalStatusCrossSiteRedirectInMainFrameNavigation PreloadPrerenderFinalStatus = "CrossSiteRedirectInMainFrameNavigation"

	// PreloadPrerenderFinalStatusCrossSiteNavigationInMainFrameNavigation enum const.
	PreloadPrerenderFinalStatusCrossSiteNavigationInMainFrameNavigation PreloadPrerenderFinalStatus = "CrossSiteNavigationInMainFrameNavigation"

	// PreloadPrerenderFinalStatusSameSiteCrossOriginRedirectNotOptInInMainFrameNavigation enum const.
	PreloadPrerenderFinalStatusSameSiteCrossOriginRedirectNotOptInInMainFrameNavigation PreloadPrerenderFinalStatus = "SameSiteCrossOriginRedirectNotOptInInMainFrameNavigation"

	// PreloadPrerenderFinalStatusSameSiteCrossOriginNavigationNotOptInInMainFrameNavigation enum const.
	PreloadPrerenderFinalStatusSameSiteCrossOriginNavigationNotOptInInMainFrameNavigation PreloadPrerenderFinalStatus = "SameSiteCrossOriginNavigationNotOptInInMainFrameNavigation"

	// PreloadPrerenderFinalStatusMemoryPressureOnTrigger enum const.
	PreloadPrerenderFinalStatusMemoryPressureOnTrigger PreloadPrerenderFinalStatus = "MemoryPressureOnTrigger"

	// PreloadPrerenderFinalStatusMemoryPressureAfterTriggered enum const.
	PreloadPrerenderFinalStatusMemoryPressureAfterTriggered PreloadPrerenderFinalStatus = "MemoryPressureAfterTriggered"

	// PreloadPrerenderFinalStatusPrerenderingDisabledByDevTools enum const.
	PreloadPrerenderFinalStatusPrerenderingDisabledByDevTools PreloadPrerenderFinalStatus = "PrerenderingDisabledByDevTools"

	// PreloadPrerenderFinalStatusSpeculationRuleRemoved enum const.
	PreloadPrerenderFinalStatusSpeculationRuleRemoved PreloadPrerenderFinalStatus = "SpeculationRuleRemoved"

	// PreloadPrerenderFinalStatusActivatedWithAuxiliaryBrowsingContexts enum const.
	PreloadPrerenderFinalStatusActivatedWithAuxiliaryBrowsingContexts PreloadPrerenderFinalStatus = "ActivatedWithAuxiliaryBrowsingContexts"

	// PreloadPrerenderFinalStatusMaxNumOfRunningEagerPrerendersExceeded enum const.
	PreloadPrerenderFinalStatusMaxNumOfRunningEagerPrerendersExceeded PreloadPrerenderFinalStatus = "MaxNumOfRunningEagerPrerendersExceeded"

	// PreloadPrerenderFinalStatusMaxNumOfRunningNonEagerPrerendersExceeded enum const.
	PreloadPrerenderFinalStatusMaxNumOfRunningNonEagerPrerendersExceeded PreloadPrerenderFinalStatus = "MaxNumOfRunningNonEagerPrerendersExceeded"

	// PreloadPrerenderFinalStatusMaxNumOfRunningEmbedderPrerendersExceeded enum const.
	PreloadPrerenderFinalStatusMaxNumOfRunningEmbedderPrerendersExceeded PreloadPrerenderFinalStatus = "MaxNumOfRunningEmbedderPrerendersExceeded"

	// PreloadPrerenderFinalStatusPrerenderingURLHasEffectiveURL enum const.
	PreloadPrerenderFinalStatusPrerenderingURLHasEffectiveURL PreloadPrerenderFinalStatus = "PrerenderingUrlHasEffectiveUrl"

	// PreloadPrerenderFinalStatusRedirectedPrerenderingURLHasEffectiveURL enum const.
	PreloadPrerenderFinalStatusRedirectedPrerenderingURLHasEffectiveURL PreloadPrerenderFinalStatus = "RedirectedPrerenderingUrlHasEffectiveUrl"

	// PreloadPrerenderFinalStatusActivationURLHasEffectiveURL enum const.
	PreloadPrerenderFinalStatusActivationURLHasEffectiveURL PreloadPrerenderFinalStatus = "ActivationUrlHasEffectiveUrl"

	// PreloadPrerenderFinalStatusJavaScriptInterfaceAdded enum const.
	PreloadPrerenderFinalStatusJavaScriptInterfaceAdded PreloadPrerenderFinalStatus = "JavaScriptInterfaceAdded"

	// PreloadPrerenderFinalStatusJavaScriptInterfaceRemoved enum const.
	PreloadPrerenderFinalStatusJavaScriptInterfaceRemoved PreloadPrerenderFinalStatus = "JavaScriptInterfaceRemoved"

	// PreloadPrerenderFinalStatusAllPrerenderingCanceled enum const.
	PreloadPrerenderFinalStatusAllPrerenderingCanceled PreloadPrerenderFinalStatus = "AllPrerenderingCanceled"
)

// PreloadPreloadingStatus Preloading status values, see also PreloadingTriggeringOutcome. This
// status is shared by prefetchStatusUpdated and prerenderStatusUpdated.
type PreloadPreloadingStatus string

const (
	// PreloadPreloadingStatusPending enum const.
	PreloadPreloadingStatusPending PreloadPreloadingStatus = "Pending"

	// PreloadPreloadingStatusRunning enum const.
	PreloadPreloadingStatusRunning PreloadPreloadingStatus = "Running"

	// PreloadPreloadingStatusReady enum const.
	PreloadPreloadingStatusReady PreloadPreloadingStatus = "Ready"

	// PreloadPreloadingStatusSuccess enum const.
	PreloadPreloadingStatusSuccess PreloadPreloadingStatus = "Success"

	// PreloadPreloadingStatusFailure enum const.
	PreloadPreloadingStatusFailure PreloadPreloadingStatus = "Failure"

	// PreloadPreloadingStatusNotSupported enum const.
	PreloadPreloadingStatusNotSupported PreloadPreloadingStatus = "NotSupported"
)

// PreloadPrefetchStatus TODO(https://crbug.com/1384419): revisit the list of PrefetchStatus and
// filter out the ones that aren't necessary to the developers.
type PreloadPrefetchStatus string

const (
	// PreloadPrefetchStatusPrefetchAllowed enum const.
	PreloadPrefetchStatusPrefetchAllowed PreloadPrefetchStatus = "PrefetchAllowed"

	// PreloadPrefetchStatusPrefetchFailedIneligibleRedirect enum const.
	PreloadPrefetchStatusPrefetchFailedIneligibleRedirect PreloadPrefetchStatus = "PrefetchFailedIneligibleRedirect"

	// PreloadPrefetchStatusPrefetchFailedInvalidRedirect enum const.
	PreloadPrefetchStatusPrefetchFailedInvalidRedirect PreloadPrefetchStatus = "PrefetchFailedInvalidRedirect"

	// PreloadPrefetchStatusPrefetchFailedMIMENotSupported enum const.
	PreloadPrefetchStatusPrefetchFailedMIMENotSupported PreloadPrefetchStatus = "PrefetchFailedMIMENotSupported"

	// PreloadPrefetchStatusPrefetchFailedNetError enum const.
	PreloadPrefetchStatusPrefetchFailedNetError PreloadPrefetchStatus = "PrefetchFailedNetError"

	// PreloadPrefetchStatusPrefetchFailedNon2XX enum const.
	PreloadPrefetchStatusPrefetchFailedNon2XX PreloadPrefetchStatus = "PrefetchFailedNon2XX"

	// PreloadPrefetchStatusPrefetchFailedPerPageLimitExceeded enum const.
	PreloadPrefetchStatusPrefetchFailedPerPageLimitExceeded PreloadPrefetchStatus = "PrefetchFailedPerPageLimitExceeded"

	// PreloadPrefetchStatusPrefetchEvictedAfterCandidateRemoved enum const.
	PreloadPrefetchStatusPrefetchEvictedAfterCandidateRemoved PreloadPrefetchStatus = "PrefetchEvictedAfterCandidateRemoved"

	// PreloadPrefetchStatusPrefetchEvictedForNewerPrefetch enum const.
	PreloadPrefetchStatusPrefetchEvictedForNewerPrefetch PreloadPrefetchStatus = "PrefetchEvictedForNewerPrefetch"

	// PreloadPrefetchStatusPrefetchHeldback enum const.
	PreloadPrefetchStatusPrefetchHeldback PreloadPrefetchStatus = "PrefetchHeldback"

	// PreloadPrefetchStatusPrefetchIneligibleRetryAfter enum const.
	PreloadPrefetchStatusPrefetchIneligibleRetryAfter PreloadPrefetchStatus = "PrefetchIneligibleRetryAfter"

	// PreloadPrefetchStatusPrefetchIsPrivacyDecoy enum const.
	PreloadPrefetchStatusPrefetchIsPrivacyDecoy PreloadPrefetchStatus = "PrefetchIsPrivacyDecoy"

	// PreloadPrefetchStatusPrefetchIsStale enum const.
	PreloadPrefetchStatusPrefetchIsStale PreloadPrefetchStatus = "PrefetchIsStale"

	// PreloadPrefetchStatusPrefetchNotEligibleBrowserContextOffTheRecord enum const.
	PreloadPrefetchStatusPrefetchNotEligibleBrowserContextOffTheRecord PreloadPrefetchStatus = "PrefetchNotEligibleBrowserContextOffTheRecord"

	// PreloadPrefetchStatusPrefetchNotEligibleDataSaverEnabled enum const.
	PreloadPrefetchStatusPrefetchNotEligibleDataSaverEnabled PreloadPrefetchStatus = "PrefetchNotEligibleDataSaverEnabled"

	// PreloadPrefetchStatusPrefetchNotEligibleExistingProxy enum const.
	PreloadPrefetchStatusPrefetchNotEligibleExistingProxy PreloadPrefetchStatus = "PrefetchNotEligibleExistingProxy"

	// PreloadPrefetchStatusPrefetchNotEligibleHostIsNonUnique enum const.
	PreloadPrefetchStatusPrefetchNotEligibleHostIsNonUnique PreloadPrefetchStatus = "PrefetchNotEligibleHostIsNonUnique"

	// PreloadPrefetchStatusPrefetchNotEligibleNonDefaultStoragePartition enum const.
	PreloadPrefetchStatusPrefetchNotEligibleNonDefaultStoragePartition PreloadPrefetchStatus = "PrefetchNotEligibleNonDefaultStoragePartition"

	// PreloadPrefetchStatusPrefetchNotEligibleSameSiteCrossOriginPrefetchRequiredProxy enum const.
	PreloadPrefetchStatusPrefetchNotEligibleSameSiteCrossOriginPrefetchRequiredProxy PreloadPrefetchStatus = "PrefetchNotEligibleSameSiteCrossOriginPrefetchRequiredProxy"

	// PreloadPrefetchStatusPrefetchNotEligibleSchemeIsNotHTTPS enum const.
	PreloadPrefetchStatusPrefetchNotEligibleSchemeIsNotHTTPS PreloadPrefetchStatus = "PrefetchNotEligibleSchemeIsNotHttps"

	// PreloadPrefetchStatusPrefetchNotEligibleUserHasCookies enum const.
	PreloadPrefetchStatusPrefetchNotEligibleUserHasCookies PreloadPrefetchStatus = "PrefetchNotEligibleUserHasCookies"

	// PreloadPrefetchStatusPrefetchNotEligibleUserHasServiceWorker enum const.
	PreloadPrefetchStatusPrefetchNotEligibleUserHasServiceWorker PreloadPrefetchStatus = "PrefetchNotEligibleUserHasServiceWorker"

	// PreloadPrefetchStatusPrefetchNotEligibleBatterySaverEnabled enum const.
	PreloadPrefetchStatusPrefetchNotEligibleBatterySaverEnabled PreloadPrefetchStatus = "PrefetchNotEligibleBatterySaverEnabled"

	// PreloadPrefetchStatusPrefetchNotEligiblePreloadingDisabled enum const.
	PreloadPrefetchStatusPrefetchNotEligiblePreloadingDisabled PreloadPrefetchStatus = "PrefetchNotEligiblePreloadingDisabled"

	// PreloadPrefetchStatusPrefetchNotFinishedInTime enum const.
	PreloadPrefetchStatusPrefetchNotFinishedInTime PreloadPrefetchStatus = "PrefetchNotFinishedInTime"

	// PreloadPrefetchStatusPrefetchNotStarted enum const.
	PreloadPrefetchStatusPrefetchNotStarted PreloadPrefetchStatus = "PrefetchNotStarted"

	// PreloadPrefetchStatusPrefetchNotUsedCookiesChanged enum const.
	PreloadPrefetchStatusPrefetchNotUsedCookiesChanged PreloadPrefetchStatus = "PrefetchNotUsedCookiesChanged"

	// PreloadPrefetchStatusPrefetchProxyNotAvailable enum const.
	PreloadPrefetchStatusPrefetchProxyNotAvailable PreloadPrefetchStatus = "PrefetchProxyNotAvailable"

	// PreloadPrefetchStatusPrefetchResponseUsed enum const.
	PreloadPrefetchStatusPrefetchResponseUsed PreloadPrefetchStatus = "PrefetchResponseUsed"

	// PreloadPrefetchStatusPrefetchSuccessfulButNotUsed enum const.
	PreloadPrefetchStatusPrefetchSuccessfulButNotUsed PreloadPrefetchStatus = "PrefetchSuccessfulButNotUsed"

	// PreloadPrefetchStatusPrefetchNotUsedProbeFailed enum const.
	PreloadPrefetchStatusPrefetchNotUsedProbeFailed PreloadPrefetchStatus = "PrefetchNotUsedProbeFailed"
)

// PreloadPrerenderMismatchedHeaders Information of headers to be displayed when the header mismatch occurred.
type PreloadPrerenderMismatchedHeaders struct {
	// HeaderName ...
	HeaderName string `json:"headerName"`

	// InitialValue (optional) ...
	InitialValue string `json:"initialValue,omitempty"`

	// ActivationValue (optional) ...
	ActivationValue string `json:"activationValue,omitempty"`
}

// PreloadEnable ...
type PreloadEnable struct{}

// ProtoReq name.
func (m PreloadEnable) ProtoReq() string { return "Preload.enable" }

// Call sends the request.
func (m PreloadEnable) Call(c Client) error {
	return call(m.ProtoReq(), m, nil, c)
}

// PreloadDisable ...
type PreloadDisable struct{}

// ProtoReq name.
func (m PreloadDisable) ProtoReq() string { return "Preload.disable" }

// Call sends the request.
func (m PreloadDisable) Call(c Client) error {
	return call(m.ProtoReq(), m, nil, c)
}

// PreloadRuleSetUpdated Upsert. Currently, it is only emitted when a rule set added.
type PreloadRuleSetUpdated struct {
	// RuleSet ...
	RuleSet *PreloadRuleSet `json:"ruleSet"`
}

// ProtoEvent name.
func (evt PreloadRuleSetUpdated) ProtoEvent() string {
	return "Preload.ruleSetUpdated"
}

// PreloadRuleSetRemoved ...
type PreloadRuleSetRemoved struct {
	// ID ...
	ID PreloadRuleSetID `json:"id"`
}

// ProtoEvent name.
func (evt PreloadRuleSetRemoved) ProtoEvent() string {
	return "Preload.ruleSetRemoved"
}

// PreloadPreloadEnabledStateUpdated Fired when a preload enabled state is updated.
type PreloadPreloadEnabledStateUpdated struct {
	// DisabledByPreference ...
	DisabledByPreference bool `json:"disabledByPreference"`

	// DisabledByDataSaver ...
	DisabledByDataSaver bool `json:"disabledByDataSaver"`

	// DisabledByBatterySaver ...
	DisabledByBatterySaver bool `json:"disabledByBatterySaver"`

	// DisabledByHoldbackPrefetchSpeculationRules ...
	DisabledByHoldbackPrefetchSpeculationRules bool `json:"disabledByHoldbackPrefetchSpeculationRules"`

	// DisabledByHoldbackPrerenderSpeculationRules ...
	DisabledByHoldbackPrerenderSpeculationRules bool `json:"disabledByHoldbackPrerenderSpeculationRules"`
}

// ProtoEvent name.
func (evt PreloadPreloadEnabledStateUpdated) ProtoEvent() string {
	return "Preload.preloadEnabledStateUpdated"
}

// PreloadPrefetchStatusUpdated Fired when a prefetch attempt is updated.
type PreloadPrefetchStatusUpdated struct {
	// Key ...
	Key *PreloadPreloadingAttemptKey `json:"key"`

	// InitiatingFrameID The frame id of the frame initiating prefetch.
	InitiatingFrameID PageFrameID `json:"initiatingFrameId"`

	// PrefetchURL ...
	PrefetchURL string `json:"prefetchUrl"`

	// Status ...
	Status PreloadPreloadingStatus `json:"status"`

	// PrefetchStatus ...
	PrefetchStatus PreloadPrefetchStatus `json:"prefetchStatus"`

	// RequestID ...
	RequestID NetworkRequestID `json:"requestId"`
}

// ProtoEvent name.
func (evt PreloadPrefetchStatusUpdated) ProtoEvent() string {
	return "Preload.prefetchStatusUpdated"
}

// PreloadPrerenderStatusUpdated Fired when a prerender attempt is updated.
type PreloadPrerenderStatusUpdated struct {
	// Key ...
	Key *PreloadPreloadingAttemptKey `json:"key"`

	// Status ...
	Status PreloadPreloadingStatus `json:"status"`

	// PrerenderStatus (optional) ...
	PrerenderStatus PreloadPrerenderFinalStatus `json:"prerenderStatus,omitempty"`

	// DisallowedMojoInterface (optional) This is used to give users more information about the name of Mojo interface
	// that is incompatible with prerender and has caused the cancellation of the attempt.
	DisallowedMojoInterface string `json:"disallowedMojoInterface,omitempty"`

	// MismatchedHeaders (optional) ...
	MismatchedHeaders []*PreloadPrerenderMismatchedHeaders `json:"mismatchedHeaders,omitempty"`
}

// ProtoEvent name.
func (evt PreloadPrerenderStatusUpdated) ProtoEvent() string {
	return "Preload.prerenderStatusUpdated"
}

// PreloadPreloadingAttemptSourcesUpdated Send a list of sources for all preloading attempts in a document.
type PreloadPreloadingAttemptSourcesUpdated struct {
	// LoaderID ...
	LoaderID NetworkLoaderID `json:"loaderId"`

	// PreloadingAttemptSources ...
	PreloadingAttemptSources []*PreloadPreloadingAttemptSource `json:"preloadingAttemptSources"`
}

// ProtoEvent name.
func (evt PreloadPreloadingAttemptSourcesUpdated) ProtoEvent() string {
	return "Preload.preloadingAttemptSourcesUpdated"
}
