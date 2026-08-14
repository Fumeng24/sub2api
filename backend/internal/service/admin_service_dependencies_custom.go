package service

// AdminServiceCustomDependencies contains site-specific collaborators kept out
// of the upstream-owned admin service constructor.
type AdminServiceCustomDependencies struct {
	gatewayService       *GatewayService
	openAIGatewayService *OpenAIGatewayService
	usageLogRepo         UsageLogRepository
	notificationEmailSvc *NotificationEmailService
	schedulerOutboxRepo  SchedulerOutboxRepository
}

func ProvideAdminServiceCustomDependencies(
	gatewayService *GatewayService,
	openAIGatewayService *OpenAIGatewayService,
	usageLogRepo UsageLogRepository,
	notificationEmailSvc *NotificationEmailService,
	schedulerOutboxRepo SchedulerOutboxRepository,
) AdminServiceCustomDependencies {
	return AdminServiceCustomDependencies{
		gatewayService:       gatewayService,
		openAIGatewayService: openAIGatewayService,
		usageLogRepo:         usageLogRepo,
		notificationEmailSvc: notificationEmailSvc,
		schedulerOutboxRepo:  schedulerOutboxRepo,
	}
}

// SetAdminServiceCustomDependencies attaches site-specific collaborators after
// the upstream constructor has initialized the service.
func SetAdminServiceCustomDependencies(service AdminService, dependencies AdminServiceCustomDependencies) {
	if impl, ok := service.(*adminServiceImpl); ok {
		impl.AdminServiceCustomDependencies = dependencies
	}
}

type AdminServiceCustomization struct{}

func ApplyAdminServiceCustomization(service AdminService, dependencies AdminServiceCustomDependencies) AdminServiceCustomization {
	SetAdminServiceCustomDependencies(service, dependencies)
	return AdminServiceCustomization{}
}
