import { Footer, Layout, Navbar } from 'nextra-theme-docs'
import { Banner, Head } from 'nextra/components'
import { getPageMap } from 'nextra/page-map'
import 'nextra-theme-docs/style.css'

export const metadata = {
    // Define your metadata here
    // For more information on metadata API, see: https://nextjs.org/docs/app/building-your-application/optimizing/metadata
}

const banner = <Banner storageKey="some-key">Steel Router is in alpha 🎉</Banner>
const navbar = (
    <Navbar
        logo={<b>STEEL ROUTER</b>}
        projectLink="https://github.com/xraph/steel"
    />
)
const footer = <Footer>Made with love {new Date().getFullYear()} © XRaph.</Footer>

export default async function RootLayout({ children }: {children: React.ReactNode }) {
    return (
        <html
            // Not required, but good for SEO
            lang="en"
            // Required to be set
            dir="ltr"
            // Suggested by `next-themes` package https://github.com/pacocoursey/next-themes#with-app
            suppressHydrationWarning
        >
        <Head
            // ... Your additional head options
        >
            {/* Your additional tags should be passed as `children` of `<Head>` element */}
        </Head>
        <body>
        <Layout
            banner={banner}
            navbar={navbar}
            pageMap={await getPageMap()}
            docsRepositoryBase="https://github.com/xraph/steel/tree/main/docs"
            footer={footer}
            darkMode={true}
            themeSwitch={{}}
        >
            {children}
        </Layout>
        </body>
        </html>
    )
}