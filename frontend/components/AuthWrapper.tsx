'use client';

import { useEffect, ReactNode } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';

interface AuthWrapperProps {
    children: ReactNode;
}

export default function AuthWrapper({ children }: AuthWrapperProps) {
    const searchParams = useSearchParams();
    const router = useRouter();

    async function getToken(code: string) {
        const clientId = process.env.NEXT_PUBLIC_COGNITO_CLIENT_ID;
        const clientSecret = process.env.NEXT_PUBLIC_COGNITO_CLIENT_SECRET;
        const domain = process.env.NEXT_PUBLIC_COGNITO_DOMAIN;

        const params = new URLSearchParams({
            grant_type: 'authorization_code',
            client_id: clientId!,
            code: code,
            redirect_uri: 'http://localhost:3000/dashboard'
        });

        const auth = Buffer.from(`${clientId}:${clientSecret}`).toString('base64');

        try {
            const response = await fetch(`${domain}/oauth2/token`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                    'Authorization': `Basic ${auth}`
                },
                body: params.toString()
            });

            const data = await response.json();
            if (data.access_token) {
                sessionStorage.setItem('access_token', data.access_token);
                return true;
            }
        } catch (error) {
            console.error('Error getting token:', error);
            return false;
        }
    }

    useEffect(() => {
        const checkAuth = async () => {
            const token = sessionStorage.getItem('access_token');
            const code = searchParams.get('code');

            if (!token && !code) {
                const cognitoDomain = process.env.NEXT_PUBLIC_COGNITO_DOMAIN;
                const clientId = process.env.NEXT_PUBLIC_COGNITO_CLIENT_ID;
                const redirectUri = encodeURIComponent('http://localhost:3000/dashboard');

                window.location.href = `${cognitoDomain}/login?response_type=code&client_id=${clientId}&redirect_uri=${redirectUri}`;
                return;
            }

            if (code) {
                const success = await getToken(code);
                if (success) {
                    router.push('/dashboard');
                }
            }
        };

        checkAuth();
    }, [searchParams]);

    return <>{children}</>;
}