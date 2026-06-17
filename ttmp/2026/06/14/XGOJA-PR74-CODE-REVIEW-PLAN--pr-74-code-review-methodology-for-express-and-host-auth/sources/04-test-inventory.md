---
Title: Targeted Test Inventory
Ticket: XGOJA-PR74-CODE-REVIEW-PLAN
Status: active
Topics:
    - review
    - goja
    - xgoja
    - auth
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Captured list of test names in review-critical packages."
LastUpdated: 2026-06-14T20:55:00-04:00
WhatFor: "Evidence captured while planning the PR 74 code review."
WhenToUse: "Use as supporting evidence for the PR 74 review methodology guide."
---

# Targeted test inventory

## ./pkg/gojahttp
- TestAccessLogResponseWriterDoesNotInventOptionalInterfaces
- TestHostStaticHandlerDoesNotSeeInventedFlusher
- TestAccessLogResponseWriterPreservesFlusherWhenUnderlyingSupportsIt
- TestAccessLogResponseWriterPreservesReaderFromAndCountsBytes
- TestValidateRoutePlanRequiresSecurityMode
- TestValidateRoutePlanPublic
- TestValidateRoutePlanUserRequiresAllowAction
- TestValidateRoutePlanResourceParamValidation
- TestValidateRoutePlanResourceDefaultsNameAndTenantParam
- TestRegisterPlannedStoresPlanOnMatchedRoute
- TestParseBodyMultipartFormFields
- TestHostRegisterHandlerPreservesRequestPath
- TestHostRegisterHandlerCanStripPrefix
- TestHostRegisterHandlerExcludePrefixes
- TestHostMountedHandlersPrecedeJSRoutes
- TestAttachHTTPHandlerHiddenRef
- TestHTTPHandlerFromValueRejectsPlainObject
- TestRouteRegistryCapturesNamedParams
- TestRouteRegistryWildcardMatchesRemainderWithoutCapture
- TestRouteRegistryWildcardMustBeSegment
- TestRegistryMatchesParamsInOrder
- TestRegistryMethodAndWildcard
- TestRegistryRoutesReturnsCopySafeDescriptors
- TestHostRoutesDelegatesToRegistry
- TestRegistryRoutesIncludesPlannedMetadata
- TestRejectRawRoutesBlocksUnplannedRoute
- TestRejectRawRoutesStillAllowsPlannedRoute
- TestPlannedPublicRouteDispatchesSecureContext
- TestPlannedUserRouteAuthenticatesAndAuthorizes
- TestPlannedUserRouteReturns401WhenUnauthenticated
- TestPlannedRouteVerifiesCSRFBeforeHandler
- TestPlannedRouteCSRFUsesIncomingMethodForAllRoutes
- TestPlannedRouteCSRFErrorBlocksHandler
- TestPlannedRouteAuditsDeniedAndCompleted
- TestPlannedRouteAuditsCSRFDenied
- TestPlannedResourceRouteResolvesAndAuthorizesResource
- TestPlannedResourceRouteMapsNotFound
- TestPlannedUserRouteMapsAuthorizerErrors
- TestSessionCookieIssuedAndReused
- ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp	0.008s

## ./modules/express
- TestExpressPlannedPublicRouteBuilder
- TestExpressGenericRouteBuilderStillWorks
- TestExpressPlannedAuthRouteBuilder
- TestExpressPlannedResourceRouteBuilder
- TestExpressPlannedBuilderSupportsCSRFAudit
- TestExpressPlannedBuilderRejectsPlainAuthObject
- TestExpressVerbHelperRejectsLegacyHandlerOverload
- TestExpressPlannedBuilderRequiresSecurityBeforeHandle
- TestExpressRouteReturnsHTMLNode
- TestExpressStaticFromAssetsModule
- TestExpressSPAFromAssetsModuleFallsBackAndExcludesAPI
- TestExpressPostJSONEcho
- TestExpressRouteAwaitsReturnedPromise
- TestExpressRouteAwaitsPromiseThatSendsResponse
- TestHeadFallsBackToGetWithoutBody
- TestExpressMountGoHTTPHandlerObject
- TestExpressMountGoHTTPHandlerCanStripPrefixAndExclude
- TestExpressMountRejectsPlainObjects
- ok  	github.com/go-go-golems/go-go-goja/modules/express	0.009s

## ./pkg/xgoja/hostauth
- TestBuildSessionManagerMapsResolvedConfig
- TestBuildAuthOptionsWiresSessionAuditResourcesAndAuthorizer
- TestServiceFactoryModeNoneBuildsNoAuthOptions
- TestServiceFactoryDevBuildsUsableAuthServices
- TestServiceFactoryUsesEnvLookupAtBuildTime
- TestLookupServiceFactory
- TestLookupServiceFactoryMissing
- TestLookupServiceFactoryRejectsWrongType
- TestLookupServiceFactoryRejectsNilTypedPointer
- TestLookupServiceFactoryRejectsMultiValueService
- TestLookupServices
- TestLookupServicesMissing
- TestLookupServicesRejectsWrongType
- TestResolveConfigDefaultsToNoAuthMemoryStoresAndSecureCookie
- TestResolveConfigParsesSessionFields
- TestResolveConfigStoreInheritanceAndEnvDSN
- TestResolveConfigExplicitMemoryStoreIgnoresInheritedDSN
- TestResolveConfigRejectsOIDCModeForThisPhase
- TestResolveConfigRejectsInvalidValuesWithPaths
- TestBuildStoresMemory
- TestBuildStoresSQLiteSharedDBAndApplySchema
- TestBuildStoresSQLiteWithoutApplySchemaDoesNotCreateTables
- TestBuildStoresPostgresConstructsWithoutConnectingWhenSchemaDisabled
- ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth	0.030s

## ./pkg/xgoja/providers/http
- TestRegister
- TestCapabilityProvidesHTTPSection
- TestCapabilityRejectsNilRuntimeInitializerHandle
- TestCapabilityDisablesHTTPWhenValuesAreNil
- TestCapabilityEnablesHTTPByDefaultWhenValuesArePresent
- TestCapabilityAllowsExplicitHTTPDisable
- TestCapabilityParsesStaticXGojaHTTPConfig
- TestCapabilityMapsExplicitGlazedHTTPConfig
- TestConfiguredInternalHostUsesDevErrorsAndRejectsRawRoutes
- TestExternalHostServiceValidation
- TestExpressProviderRegistersIntoExternalHost
- TestExpressProviderRegistersPlannedPublicRouteIntoExternalHost
- TestExpressExternalHostDoesNotBindConfiguredHTTPPort
- TestExpressRequireDoesNotBindHTTPPort
- TestCapabilityStartReportsPortConflictsSynchronously
- TestNewServeCommandSetRequiresJSVerbSources
- TestNewServeCommandSetBuildsVerbCommandsWithHTTPSection
- TestNewServeCommandSetRejectsMalformedHostAuthServiceFactory
- TestServeVerbLoadsIncludedHelperModulesWithoutHelperCommands
- TestServeVerbUsesHostAuthServiceFactory
- TestServeVerbPreservesExternalHostWithHostAuthFactory
- TestHostOptionsWithAuthPreservesHTTPSettings
- TestServeVerbHotReloadServesStatusAndReloadsChangedSource
- TestServeVerbHotReloadUsesHostAuthServiceFactory
- TestAppendTypeScriptWatchExtensions
- TestSourceSetHasTypeScript
- ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.029s

## ./pkg/gojahttp/auth/sessionauth
- TestAuthenticateAndCSRF
- TestAuthenticateFailures
- TestAuthenticateRequiresFreshMFA
- TestMemoryStoreRotateValidatesNextBeforeDeletingOld
- TestExpiredRevokedAndRotatedSessions
- TestCSRFMismatchAndCookieClearing
- TestCustomActorLoader
- TestMemoryStoreContract
- ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth	0.004s

## ./pkg/gojahttp/auth/audit
- TestNormalizeRecordAndRedaction
- TestMemorySinkAndStoreSink
- TestLogSinkOmitsSensitiveRequestMetadata
- TestMemoryStoreContract
- ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit	0.006s

## ./pkg/gojahttp/auth/capability
- TestIssueRedeemSingleUseAndAudit
- TestRedeemFailures
- TestRevoke
- TestOrgInviteFlow
- TestIssueValidation
- TestMemoryStoreContract
- ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability	0.004s

## ./pkg/gojahttp/auth/appauth
- TestResourceResolver
- TestAuthorizerAllowsExpectedActions
- TestAuthorizerDeniesNegativeCases
- TestUpsertFromOIDCRejectsDisabledExistingUser
- TestUserAndMembershipStore
- TestAuthorizerPropagatesBackendErrors
- TestMemoryStoreContract
- ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth	0.004s

## ./pkg/gojahttp/auth/keycloakauth
- TestLoginCallbackCreatesSession
- TestCallbackRejectsBadStateAndNonce
- TestCallbackRejectsExpiredTokenAndWrongAudience
- TestCallbackRejectsNormalizerFailureAndLogoutClearsCookie
- ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth	0.004s

