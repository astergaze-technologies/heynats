import { ChevronDown, ChevronRight, Copy } from 'lucide-react';
import { useState } from 'react';
import { isLosslessNumber } from '@/lib/json';

// JSON Recursive Node Component
interface JsonNodeProps {
  name?: string;
  value: unknown;
  isLast: boolean;
  defaultExpanded?: boolean;
}

function JsonNode({
  name,
  value,
  isLast,
  defaultExpanded = false,
}: JsonNodeProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  // Helper to render property key
  const renderKey = () => {
    if (!name) return null;
    return (
      <span className="mr-1">
        <span className="text-primary">"{name}"</span>
        <span className="text-muted-foreground">:</span>
      </span>
    );
  };

  // Helper to render trailing comma
  const renderComma = () => {
    if (!isLast) return <span className="text-muted-foreground">,</span>;
    return null;
  };

  if (value === null) {
    return (
      <div className="font-mono text-sm leading-6">
        {renderKey()}
        <span className="text-muted-foreground">null</span>
        {renderComma()}
      </div>
    );
  }

  if (typeof value === 'boolean') {
    return (
      <div className="font-mono text-sm leading-6">
        {renderKey()}
        <span className="text-warning">{value.toString()}</span>
        {renderComma()}
      </div>
    );
  }

  if (typeof value === 'number' || isLosslessNumber(value)) {
    return (
      <div className="font-mono text-sm leading-6">
        {renderKey()}
        <span className="text-primary">{String(value)}</span>
        {renderComma()}
      </div>
    );
  }

  if (typeof value === 'string') {
    return (
      <div className="font-mono text-sm leading-6">
        {renderKey()}
        <span className="text-success">"{value}"</span>
        {renderComma()}
      </div>
    );
  }

  // Arrays and Objects
  if (typeof value === 'object') {
    const isArray = Array.isArray(value);
    const keys = Object.keys(value as object);
    const isEmpty = keys.length === 0;
    const openChar = isArray ? '[' : '{';
    const closeChar = isArray ? ']' : '}';
    const itemCount = keys.length;

    if (isEmpty) {
      return (
        <div className="font-mono text-sm leading-6">
          {renderKey()}
          <span className="text-muted-foreground">
            {openChar}
            {closeChar}
          </span>
          {renderComma()}
        </div>
      );
    }

    return (
      <div className="font-mono text-sm leading-6">
        <div className="flex items-start">
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              setExpanded(!expanded);
            }}
            className="mr-1 mt-1 p-0.5 hover:bg-muted rounded focus:outline-none focus:ring-1 focus:ring-border"
          >
            {expanded ? (
              <ChevronDown className="w-3 h-3 text-muted-foreground" />
            ) : (
              <ChevronRight className="w-3 h-3 text-muted-foreground" />
            )}
          </button>

          <div className="flex-1">
            <span>
              {renderKey()}
              <span className="text-muted-foreground">{openChar}</span>
            </span>

            {!expanded && (
              <>
                <button
                  type="button"
                  onClick={() => setExpanded(true)}
                  className="px-1 text-muted-foreground hover:text-muted-foreground text-xs bg-muted rounded mx-1"
                >
                  {itemCount} {itemCount === 1 ? 'item' : 'items'}
                </button>
                <span className="text-muted-foreground">{closeChar}</span>
                {renderComma()}
              </>
            )}
          </div>
        </div>

        {expanded && (
          <div>
            <div className="pl-6 border-l border-border ml-2.5">
              {keys.map((key, index) => (
                <JsonNode
                  key={key}
                  name={isArray ? undefined : key}
                  value={(value as Record<string, unknown>)[key]}
                  isLast={index === keys.length - 1}
                  defaultExpanded={false}
                />
              ))}
            </div>
            <div className="ml-5">
              <span className="text-muted-foreground">{closeChar}</span>
              {renderComma()}
            </div>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="font-mono text-sm leading-6">
      {renderKey()}
      <span className="text-foreground">{String(value)}</span>
      {renderComma()}
    </div>
  );
}

// JSON Viewer Component
export function JsonViewer({ data }: { data: unknown }) {
  // We can treat the root as a "value" with no name and isLast=true
  return (
    <div className="w-full">
      <JsonNode value={data} isLast={true} defaultExpanded={true} />
    </div>
  );
}

// Code Display Component with Copy
interface CodeDisplayProps {
  code: string;
  maxHeight?: string;
}

export function CodeDisplay({
  code,
  maxHeight = 'max-h-64',
}: CodeDisplayProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative">
      <button
        onClick={handleCopy}
        className="absolute top-2 right-2 p-2 bg-muted hover:bg-muted rounded transition-colors z-10"
        title="Copy to clipboard"
        type="button"
      >
        <Copy className="w-4 h-4" />
      </button>
      {copied && (
        <div className="absolute top-2 right-12 px-2 py-1 bg-success text-success-foreground text-xs rounded">
          Copied!
        </div>
      )}
      <pre
        className={`p-3 bg-muted text-foreground rounded border border-input font-mono text-sm overflow-auto ${maxHeight}`}
      >
        <code>{code}</code>
      </pre>
    </div>
  );
}
