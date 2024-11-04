import '@/app/ui/global.css';
import { inter } from '@/app/ui/components/fonts';
import { Toaster } from '@/components/ui/toaster';

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={`${inter.className} antialiased`}>{children}
        <Toaster />
      </body>
    </html>
  );
}
