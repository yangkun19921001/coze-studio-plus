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

import type { HTMLAttributes, ReactNode } from 'react';

import classNames from 'classnames';

import './code-block.less';

interface HastElementLike {
  tagName?: string;
  [key: string]: unknown;
}

type UnistPositionLike = {
  start?: { line?: number; column?: number };
  end?: { line?: number; column?: number };
} | null;

type CodeBlockProps = HTMLAttributes<HTMLElement> & {
  inline?: boolean;
  children?: ReactNode;
  node?: HastElementLike;
  sourcePosition?: UnistPositionLike;
  index?: number;
  siblingCount?: number;
};

/**
 * Code block component
 * Handles both inline code and code blocks with syntax highlighting
 */
export const CodeBlock = ({
  inline,
  className,
  children,
  ...props
}: CodeBlockProps) => {
  const match = /language-(\w+)/.exec(className || '');
  const language = match ? match[1] : '';

  // 判断是否为内联代码：
  // react-markdown 10.x 中，inline prop 明确标识内联代码
  // 内联代码：inline === true
  // 代码块：inline === false 或 undefined，且有 language
  const isInline = inline === true;

  if (isInline) {
    return (
      <code
        className={classNames('coze-markdown-code-inline', className)}
        style={{
          display: 'inline',
          whiteSpace: 'nowrap',
          wordBreak: 'normal',
        }}
        {...props}
      >
        {children}
      </code>
    );
  }

  if (!language) {
    return (
      <pre className={classNames('coze-markdown-code-block-pre', className)}>
        <code className={classNames(className)} {...props}>
          {children}
        </code>
      </pre>
    );
  }

  return (
    <div className="coze-markdown-code-block">
      {language ? (
        <div className="coze-markdown-code-block-lang">{language}</div>
      ) : null}
      <pre className={classNames('coze-markdown-code-block-pre', className)}>
        <code className={classNames(className)} {...props}>
          {children}
        </code>
      </pre>
    </div>
  );
};

CodeBlock.displayName = 'CodeBlock';
