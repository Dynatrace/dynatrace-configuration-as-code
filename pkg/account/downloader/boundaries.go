/*
 * @license
 * Copyright 2025 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package downloader

import (
	"context"
	"fmt"

	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	stringutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

type (
	Boundaries []boundary

	boundary struct {
		boundary *account.Boundary
		dto      *accountmanagement.PolicyBoundaryOverview
	}
)

func (a *Downloader) boundaries(ctx context.Context) (Boundaries, error) {
	log.InfoContext(ctx, "Downloading boundaries")
	dtos, err := a.httpClient.GetBoundaries(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of boundaries for account %q from DT: %w", a.accountInfo, err)
	}
	log.DebugContext(ctx, "Downloaded %d boundaries", len(dtos))

	retVal := make(Boundaries, 0, len(dtos))
	for i := range dtos {
		b := toAccountBoundary(&dtos[i])

		retVal = append(retVal, boundary{
			boundary: b,
			dto:      &dtos[i],
		})
	}

	log.InfoContext(ctx, "Downloaded %d boundaries", len(retVal.asAccountBoundaries()))
	return retVal, nil
}

func toAccountBoundary(dto *accountmanagement.PolicyBoundaryOverview) *account.Boundary {
	return &account.Boundary{
		ID:             stringutils.Sanitize(dto.Name),
		Name:           dto.Name,
		Query:          dto.BoundaryQuery,
		OriginObjectID: dto.Uuid,
	}
}

func (b Boundaries) asAccountBoundaries() map[account.BoundaryId]account.Boundary {
	retVal := make(map[account.BoundaryId]account.Boundary)
	for i := range b {
		retVal[b[i].boundary.ID] = *b[i].boundary
	}
	return retVal
}

func (b Boundaries) RefOn(boundaryUUIDs ...string) []account.Ref {
	var retVal []account.Ref
	for _, bnd := range b {
		for _, uuid := range boundaryUUIDs {
			if bnd.dto.Uuid == uuid {
				retVal = append(retVal, bnd.RefOn())
				break
			}
		}
	}
	return retVal
}

func (b *boundary) RefOn() account.Ref {
	return account.Reference{Id: b.boundary.ID}
}
