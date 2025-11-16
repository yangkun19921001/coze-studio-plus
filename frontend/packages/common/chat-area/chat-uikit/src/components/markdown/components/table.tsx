/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { type FC, type HTMLAttributes } from 'react';

import './table.less';

/**
 * Table wrapper component
 * Provides responsive scrolling for wide tables
 */
export const Table: FC<HTMLAttributes<HTMLTableElement>> = ({
  children,
  ...props
}) => (
  <div className="coze-markdown-table-wrapper">
    <table className="coze-markdown-table" {...props}>
      {children}
    </table>
  </div>
);

/**
 * Table head cell component
 */
export const TableHead: FC<HTMLAttributes<HTMLTableCellElement>> = ({
  children,
  ...props
}) => (
  <th className="coze-markdown-table-th" {...props}>
    {children}
  </th>
);

/**
 * Table data cell component
 */
export const TableCell: FC<HTMLAttributes<HTMLTableCellElement>> = ({
  children,
  ...props
}) => (
  <td className="coze-markdown-table-td" {...props}>
    {children}
  </td>
);

Table.displayName = 'Table';
TableHead.displayName = 'TableHead';
TableCell.displayName = 'TableCell';
