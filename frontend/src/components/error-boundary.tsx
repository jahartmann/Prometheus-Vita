"use client";

import React from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("ErrorBoundary caught:", error, errorInfo);
  }

  render() {
    if (!this.state.hasError) {
      return this.props.children;
    }

    if (this.props.fallback) {
      return this.props.fallback;
    }

    return (
      <Card className="m-4">
        <CardContent className="flex flex-col items-center gap-3 py-6">
          <AlertTriangle className="h-8 w-8 text-destructive" />
          <div className="text-center">
            <p className="text-sm font-medium">Komponente konnte nicht geladen werden</p>
            <p className="mt-1 text-xs text-muted-foreground">
              {this.state.error?.message || "Ein unerwarteter Fehler ist aufgetreten."}
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => this.setState({ hasError: false, error: null })}
          >
            <RefreshCw className="mr-2 h-3 w-3" />
            Erneut versuchen
          </Button>
        </CardContent>
      </Card>
    );
  }
}
