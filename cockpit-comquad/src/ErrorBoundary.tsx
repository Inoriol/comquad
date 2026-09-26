import React, { Component, ErrorInfo, ReactNode } from 'react';
import { Alert, Page, PageSection, Button } from '@patternfly/react-core';
import { _ } from './i18n';

interface Props {
    children: ReactNode;
    fallback?: ReactNode;
}

interface State {
    hasError: boolean;
    error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
    public state: State = {
        hasError: false,
        error: null
    };

    public static getDerivedStateFromError (error: Error): State {
        return { hasError: true, error };
    }

    public componentDidCatch (error: Error, errorInfo: ErrorInfo) {
        console.error('ErrorBoundary caught an error:', error, errorInfo);
    }

    public render () {
        if (this.state.hasError) {
            if (this.props.fallback) {
                return this.props.fallback;
            }

            return (
                <Page>
                    <PageSection>
                        <Alert
                            variant='danger'
                            title={_('Something went wrong')}
                            actionClose={
                                <Button
                                    variant='link'
                                    onClick={() => this.setState({ hasError: false, error: null })}
                              >
                                    {_('Try again')}
                              </Button>
                            }
                      >
                            {this.state.error?.message || _('An unexpected error occurred')}
                      </Alert>
                  </PageSection>
              </Page>
            );
        }

        return this.props.children;
    }
}
