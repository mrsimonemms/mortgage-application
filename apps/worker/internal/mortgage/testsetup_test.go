package mortgage

import (
	"github.com/mrsimonemms/mortgage-application/mortgage-application/apps/worker/internal/mortgage/activities"
)

func init() {
	activities.DisableActivityDelaysForTests()
}
