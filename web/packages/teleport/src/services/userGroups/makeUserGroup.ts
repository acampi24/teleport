/**
 * Copyright 2023 Gravitational, Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { UserGroup } from './types';

export function makeUserGroup(json): UserGroup {
  const { name, description, friendlyName } = json;

  const labels = json.labels || [];
  const applications = json.applications || [];

  return {
    kind: 'user_group',
    name,
    description,
    labels,
    friendlyName,
    applications,
  };
}
