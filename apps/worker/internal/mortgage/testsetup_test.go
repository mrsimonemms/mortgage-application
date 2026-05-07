package mortgage

import (
	"github.com/temporal-sa/mortgage-application-demo/apps/worker/internal/mortgage/activities"
)

func init() {
	activities.DisableActivityDelaysForTests()
}
