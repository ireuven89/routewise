import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import {
    FaBriefcase,
    FaCalendarCheck,
    FaUsers,
    FaPlus,
    FaClock,
    FaCopy,
    FaCheck,
    FaHardHat,
    FaWrench,
    FaCheckCircle,
    FaPhone,
    FaChartBar,
    FaArrowRight,
} from 'react-icons/fa';
import {
    BarChart,
    Bar,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    ResponsiveContainer,
} from 'recharts';
import Layout from '../components/Layout';
import { StatCardSkeleton, CardSkeleton } from '../components/Skeleton';
import { customersAPI, dashboardAPI, jobsAPI, workersAPI } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import ServiceCallModal from '../components/ServiceCallModal';
import JobModal from '../components/JobModal';
import CustomerModal from '../components/CustomerModal';
import WorkerModal from '../components/WorkerModal';

// ─── Dashboard ───────────────────────────────────────────────────────────────
const Dashboard = () => {
    const { organization } = useAuth();
    const { t } = useLanguage();
    const industry = organization?.industry || 'hvac';
    const isConstruction = industry === 'construction';

    const industryT = {
        workers: t(`industry.${industry}.workers`),
        workerSingle: t(`industry.${industry}.workerSingle`),
        jobs: t(`industry.${industry}.jobs`),
        jobSingle: t(`industry.${industry}.jobSingle`),
        newJob: t(`industry.${industry}.newJob`),
        addWorker: t(`industry.${industry}.addWorker`),
        registerWorker: t(`industry.${industry}.registerWorker`),
        icon: isConstruction ? FaHardHat : FaWrench,
    };

    const [stats, setStats] = useState({
        totalJobs: 0,
        scheduledJobs: 0,
        activeJobs: 0,
        completedJobs: 0,
        totalCustomers: 0,
        totalWorkers: 0,
    });
    const [todayJobs, setTodayJobs] = useState([]);
    const [workers, setWorkers] = useState([]);
    const [allWorkers, setAllWorkers] = useState([]);
    const [customers, setCustomers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [revenueStats, setRevenueStats] = useState(null);
    const [revenueLoading, setRevenueLoading] = useState(true);
    const [copied, setCopied] = useState(false);
    const [showServiceCallModal, setShowServiceCallModal] = useState(false);
    const [showJobModal, setShowJobModal] = useState(false);
    const [showCustomerModal, setShowCustomerModal] = useState(false);
    const [showWorkerModal, setShowWorkerModal] = useState(false);

    const fetchDashboardData = useCallback(async () => {
        setLoading(true);
        setRevenueLoading(true);
        try {
            const [jobsRes, customersRes, workersRes, statsRes] = await Promise.all([
                jobsAPI.getAll(),
                customersAPI.getAll(),
                workersAPI.getAll(false),
                dashboardAPI.getStats(),
            ]);

            const jobs = jobsRes.data || [];
            const allWorkers = workersRes.data || [];

            const scheduledJobs = jobs.filter((j) => j.status === 'scheduled');
            const activeJobs = jobs.filter((j) => j.status === 'in_progress');
            const completedJobs = jobs.filter((j) => j.status === 'completed');

            if (!isConstruction) {
                const now = new Date();
                const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0);
                const todayEnd = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59);
                setTodayJobs(
                    jobs
                        .filter((j) => {
                            const d = new Date(j.scheduled_at);
                            return d >= todayStart && d <= todayEnd;
                        })
                        .slice(0, 5)
                );
            }

            setStats({
                totalJobs: jobs.length,
                scheduledJobs: scheduledJobs.length,
                activeJobs: activeJobs.length,
                completedJobs: completedJobs.length,
                totalCustomers: customersRes.data?.length || 0,
                totalWorkers: allWorkers.length,
            });
            setAllWorkers(allWorkers);
            setWorkers(allWorkers.slice(0, 7));
            setCustomers(customersRes.data || []);
            setRevenueStats(statsRes.data?.revenue || null);
        } catch (error) {
            console.error('Error fetching dashboard data:', error);
        } finally {
            setLoading(false);
            setRevenueLoading(false);
        }
    }, [isConstruction]);

    useEffect(() => {
        fetchDashboardData();
    }, [fetchDashboardData]);

    const handleCopyCode = () => {
        if (!organization?.company_code) return;
        navigator.clipboard.writeText(organization.company_code).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        });
    };

    const handleCreateJob = async (jobData) => {
        try {
            await jobsAPI.create(jobData);
            setShowJobModal(false);
            fetchDashboardData();
        } catch (error) {
            console.error('Failed to create job:', error);
        }
    };

    const handleCreateCustomer = async (customerData) => {
        try {
            await customersAPI.create(customerData);
            setShowCustomerModal(false);
            fetchDashboardData();
        } catch (error) {
            console.error('Failed to create customer:', error);
        }
    };

    const handleCreateWorker = async (workerData) => {
        try {
            await workersAPI.create(workerData);
            setShowWorkerModal(false);
            fetchDashboardData();
        } catch (error) {
            console.error('Failed to create worker:', error);
        }
    };

    const todayLabel = new Date().toLocaleDateString('en-US', {
        weekday: 'long',
        month: 'long',
        day: 'numeric',
        year: 'numeric',
    });

    return (
        <Layout>
            <div className="min-h-screen bg-gray-50">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">

                    {/* ── Hero Header ─────────────────────────────────────────── */}
                    <div
                        className="rounded-2xl shadow-xl overflow-hidden"
                        style={{ background: 'linear-gradient(135deg, #1e3a5f 0%, #2d5a8e 60%, #1e4d7b 100%)' }}
                    >
                        <div className="px-6 py-6 sm:px-8 sm:py-7">
                            <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                                {/* Left: greeting */}
                                <div>
                                    <p className="text-blue-300 text-sm font-medium">{todayLabel}</p>
                                    <h1 className="text-2xl sm:text-3xl font-bold text-white mt-1">
                                        {t('dashboard.welcome')} {organization?.name || ''}
                                    </h1>
                                    <div className="flex items-center gap-2 mt-2">
                                        <span className="inline-flex items-center gap-1.5 bg-white bg-opacity-15 text-blue-100 text-xs font-semibold px-3 py-1 rounded-full">
                                            <industryT.icon className="w-3 h-3" />
                                            {industry.toUpperCase()}
                                        </span>
                                    </div>
                                </div>

                                {/* Right: actions */}
                                <div className="flex flex-col gap-3 sm:items-end">
                                    <div className="flex items-center gap-2">
                                        <button
                                            onClick={() => setShowServiceCallModal(true)}
                                            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold text-white shadow-md hover:shadow-lg transition-all duration-200 hover:-translate-y-0.5"
                                            style={{ backgroundColor: '#ff6b35' }}
                                        >
                                            <FaPhone className="w-3.5 h-3.5" />
                                            {t('dashboard.newServiceCall')}
                                        </button>
                                        <Link
                                            to="/jobs"
                                            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold bg-white bg-opacity-15 text-white hover:bg-opacity-25 transition-all duration-200"
                                        >
                                            <FaPlus className="w-3.5 h-3.5" />
                                            {industryT.newJob}
                                        </Link>
                                    </div>

                                    {/* Company code */}
                                    {organization?.company_code && (
                                        <div className="flex items-center gap-2 bg-white bg-opacity-10 border border-white border-opacity-20 rounded-xl px-4 py-2">
                                            <div>
                                                <p className="text-blue-300 text-xs">{t('dashboard.companyCodeLabel')}</p>
                                                <span className="text-white font-mono font-bold tracking-widest text-base">
                                                    {organization.company_code}
                                                </span>
                                            </div>
                                            <button
                                                onClick={handleCopyCode}
                                                className="ml-2 flex items-center gap-1 bg-white bg-opacity-20 hover:bg-opacity-30 text-white text-xs font-medium px-2.5 py-1 rounded-lg transition-all"
                                            >
                                                {copied ? <FaCheck className="w-3 h-3" /> : <FaCopy className="w-3 h-3" />}
                                                {copied ? t('dashboard.copied') : t('dashboard.copy')}
                                            </button>
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* ── Stat Cards ──────────────────────────────────────────── */}
                    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                        {loading ? (
                            <>
                                <StatCardSkeleton /><StatCardSkeleton />
                                <StatCardSkeleton /><StatCardSkeleton />
                            </>
                        ) : isConstruction ? (
                            <>
                                <StatCard title={t('dashboard.activeProjects')} value={stats.activeJobs} sub={`${stats.scheduledJobs} ${t('dashboard.scheduled')}`} link="/jobs" icon={FaBriefcase} color="blue" />
                                <StatCard title={t('dashboard.completed')} value={stats.completedJobs} sub={t('dashboard.thisQuarter')} link="/jobs?status=completed" icon={FaCheckCircle} color="emerald" />
                                <StatCard title={t('dashboard.totalProjects')} value={stats.totalJobs} sub={t('dashboard.allTime')} link="/jobs" icon={FaCalendarCheck} color="violet" />
                                <StatCard title={industryT.workers} value={stats.totalWorkers} link="/technicians" icon={FaHardHat} color="amber" />
                            </>
                        ) : (
                            <>
                                <StatCard title={t('dashboard.totalJobs')} value={stats.totalJobs} link="/jobs" icon={FaBriefcase} color="blue" />
                                <StatCard title={t('dashboard.scheduledToday')} value={stats.scheduledJobs} link="/jobs?status=scheduled" icon={FaCalendarCheck} color="emerald" />
                                <StatCard title={t('dashboard.customers')} value={stats.totalCustomers} link="/customers" icon={FaUsers} color="violet" />
                                <StatCard title={industryT.workers} value={stats.totalWorkers} link="/technicians" icon={FaWrench} color="amber" />
                            </>
                        )}
                    </div>

                    {/* ── Revenue Board ───────────────────────────────────────── */}
                    {revenueLoading ? (
                        <CardSkeleton />
                    ) : (
                        <RevenueBoardPanel revenueStats={revenueStats} t={t} />
                    )}

                    {/* ── Main Panels ─────────────────────────────────────────── */}
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                        <div className="lg:col-span-2">
                            {loading ? (
                                <CardSkeleton />
                            ) : isConstruction ? (
                                <ActiveWorkersPanel workers={workers} industryT={industryT} t={t} />
                            ) : (
                                <TodaySchedulePanel todayJobs={todayJobs} t={t} />
                            )}
                        </div>
                        <div className="lg:col-span-1">
                            {loading ? (
                                <CardSkeleton />
                            ) : (
                                <QuickActionsPanel
                                    t={t}
                                    industryT={industryT}
                                    onServiceCall={() => setShowServiceCallModal(true)}
                                    onNewJob={() => setShowJobModal(true)}
                                    onAddCustomer={() => setShowCustomerModal(true)}
                                    onAddWorker={() => setShowWorkerModal(true)}
                                />
                            )}
                        </div>
                    </div>
                </div>
            </div>

            {/* ── Modals ────────────────────────────────────────────────── */}
            {showServiceCallModal && (
                <ServiceCallModal
                    workers={allWorkers}
                    onSuccess={() => { setShowServiceCallModal(false); fetchDashboardData(); }}
                    onClose={() => setShowServiceCallModal(false)}
                />
            )}
            {showJobModal && (
                <JobModal
                    customers={customers}
                    technicians={allWorkers}
                    onSave={handleCreateJob}
                    onClose={() => setShowJobModal(false)}
                />
            )}
            {showCustomerModal && (
                <CustomerModal
                    onSave={handleCreateCustomer}
                    onClose={() => setShowCustomerModal(false)}
                />
            )}
            {showWorkerModal && (
                <WorkerModal
                    onSave={handleCreateWorker}
                    onClose={() => setShowWorkerModal(false)}
                />
            )}
        </Layout>
    );
};

// ─── StatCard ─────────────────────────────────────────────────────────────────
const statThemes = {
    blue:    { circle: 'bg-blue-100',    icon: 'text-blue-600',    value: 'text-blue-600',    ring: 'ring-blue-100' },
    emerald: { circle: 'bg-emerald-100', icon: 'text-emerald-600', value: 'text-emerald-600', ring: 'ring-emerald-100' },
    violet:  { circle: 'bg-violet-100',  icon: 'text-violet-600',  value: 'text-violet-600',  ring: 'ring-violet-100' },
    amber:   { circle: 'bg-amber-100',   icon: 'text-amber-600',   value: 'text-amber-600',   ring: 'ring-amber-100' },
};

const StatCard = ({ title, value, sub, link, icon: Icon, color }) => {
    const theme = statThemes[color] || statThemes.blue;
    return (
        <Link
            to={link}
            className="group bg-white rounded-2xl shadow-sm border border-gray-100 p-5 hover:shadow-md transition-all duration-200 hover:-translate-y-0.5 flex flex-col gap-4"
        >
            <div className={`w-11 h-11 rounded-full ${theme.circle} flex items-center justify-center ring-4 ${theme.ring}`}>
                <Icon className={`w-5 h-5 ${theme.icon}`} />
            </div>
            <div>
                <p className={`text-3xl font-bold ${theme.value}`}>{value}</p>
                <p className="text-sm font-medium text-gray-600 mt-1">{title}</p>
                {sub && <p className="text-xs text-gray-400 mt-0.5">{sub}</p>}
            </div>
        </Link>
    );
};

// ─── Revenue Board Panel ──────────────────────────────────────────────────────
const fmt = (val) =>
    val == null ? '—' : Number(val).toLocaleString('en-IL', { maximumFractionDigits: 0 });

const RevenueMetricCard = ({ label, value, currency, highlight }) => (
    <div className={`rounded-2xl p-5 flex flex-col gap-2 ${highlight
        ? 'bg-gradient-to-br from-emerald-500 to-emerald-600 text-white shadow-md'
        : 'bg-gray-50 border border-gray-100'}`}
    >
        <p className={`text-xs font-semibold uppercase tracking-wider ${highlight ? 'text-emerald-100' : 'text-gray-500'}`}>
            {label}
        </p>
        <p className={`text-2xl font-bold ${highlight ? 'text-white' : 'text-gray-900'}`}>
            {currency}{fmt(value)}
        </p>
    </div>
);

const RevenueBoardPanel = ({ revenueStats, t }) => {
    const currency = t('dashboard.revenue.currency');
    const chartData = (revenueStats?.revenue_by_month || []).map((m) => ({
        name: m.month,
        revenue: m.revenue,
    }));

    return (
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
            <div className="px-6 py-5 border-b border-gray-100 flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-full bg-emerald-100 flex items-center justify-center">
                    <FaChartBar className="w-3.5 h-3.5 text-emerald-600" />
                </div>
                <h2 className="text-base font-semibold text-gray-900">{t('dashboard.revenue.title')}</h2>
            </div>
            <div className="p-6">
                <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
                    <RevenueMetricCard label={t('dashboard.revenue.thisMonth')} value={revenueStats?.this_month} currency={currency} highlight />
                    <RevenueMetricCard label={t('dashboard.revenue.thisWeek')} value={revenueStats?.this_week} currency={currency} />
                    <RevenueMetricCard label={t('dashboard.revenue.totalRevenue')} value={revenueStats?.total} currency={currency} />
                    <RevenueMetricCard label={t('dashboard.revenue.avgJobValue')} value={revenueStats?.avg_job_value} currency={currency} />
                </div>
                {chartData.length > 0 ? (
                    <>
                        <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-4">
                            {t('dashboard.revenue.monthlyTrend')}
                        </p>
                        <ResponsiveContainer width="100%" height={200}>
                            <BarChart data={chartData} margin={{ top: 0, right: 4, left: 0, bottom: 0 }}>
                                <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" vertical={false} />
                                <XAxis dataKey="name" tick={{ fontSize: 11, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                                <YAxis tick={{ fontSize: 11, fill: '#9ca3af' }} axisLine={false} tickLine={false} tickFormatter={(v) => `${currency}${fmt(v)}`} />
                                <Tooltip
                                    formatter={(value) => [`${currency}${fmt(value)}`, t('dashboard.revenue.totalRevenue')]}
                                    contentStyle={{ fontSize: 12, borderRadius: 12, border: 'none', boxShadow: '0 4px 20px rgba(0,0,0,0.1)' }}
                                    cursor={{ fill: '#f0fdf4' }}
                                />
                                <Bar dataKey="revenue" fill="#10b981" radius={[6, 6, 0, 0]} maxBarSize={48} />
                            </BarChart>
                        </ResponsiveContainer>
                    </>
                ) : (
                    <div className="py-8 text-center">
                        <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
                            <FaChartBar className="w-5 h-5 text-gray-300" />
                        </div>
                        <p className="text-sm text-gray-400">{t('dashboard.revenue.noData')}</p>
                    </div>
                )}
            </div>
        </div>
    );
};

// ─── Today's Schedule Panel ───────────────────────────────────────────────────
const TodaySchedulePanel = ({ todayJobs, t }) => (
    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden h-full">
        <div className="px-6 py-5 border-b border-gray-100 flex items-center justify-between">
            <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center">
                    <FaClock className="w-3.5 h-3.5 text-blue-600" />
                </div>
                <h2 className="text-base font-semibold text-gray-900">{t('dashboard.todaySchedule')}</h2>
            </div>
            <Link to="/jobs" className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 hover:text-blue-700">
                {t('dashboard.viewAll')} <FaArrowRight className="w-2.5 h-2.5" />
            </Link>
        </div>

        {todayJobs.length === 0 ? (
            <div className="px-6 py-12 text-center">
                <div className="w-14 h-14 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                    <FaCalendarCheck className="w-6 h-6 text-gray-300" />
                </div>
                <p className="text-gray-500 text-sm font-medium">{t('dashboard.noJobsToday')}</p>
                <Link
                    to="/jobs"
                    className="mt-4 inline-flex items-center gap-1.5 text-sm font-semibold text-blue-600 hover:text-blue-700"
                >
                    <FaPlus className="w-3 h-3" />
                    {t('dashboard.scheduleAJob')}
                </Link>
            </div>
        ) : (
            <div className="divide-y divide-gray-50">
                {todayJobs.map((job) => (
                    <Link
                        key={job.id}
                        to={`/jobs/${job.id}`}
                        className="flex items-center px-6 py-4 hover:bg-gray-50 transition-colors duration-100 group"
                    >
                        <div className="flex-shrink-0 mr-4">
                            <span className="inline-block text-xs font-bold text-blue-600 bg-blue-50 border border-blue-100 px-2.5 py-1 rounded-lg min-w-[52px] text-center">
                                {new Date(job.scheduled_at).toLocaleTimeString('en-US', {
                                    hour: '2-digit',
                                    minute: '2-digit',
                                })}
                            </span>
                        </div>
                        <div className="flex-1 min-w-0">
                            <p className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors truncate">
                                {job.title}
                            </p>
                            {job.description && (
                                <p className="text-xs text-gray-400 truncate mt-0.5">{job.description}</p>
                            )}
                        </div>
                        <div className="flex-shrink-0 ml-3">
                            <StatusBadge status={job.status} t={t} />
                        </div>
                    </Link>
                ))}
            </div>
        )}
    </div>
);

// ─── Active Workers Panel ─────────────────────────────────────────────────────
const getWorkerStatus = (worker) => (!worker.is_active ? 'offline' : 'available');

const ActiveWorkersPanel = ({ workers, industryT, t }) => (
    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden h-full">
        <div className="px-6 py-5 border-b border-gray-100 flex items-center justify-between">
            <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-full bg-amber-100 flex items-center justify-center">
                    <FaHardHat className="w-3.5 h-3.5 text-amber-600" />
                </div>
                <h2 className="text-base font-semibold text-gray-900">
                    {t('dashboard.activeWorkers', { workers: industryT.workers })}
                </h2>
            </div>
            <Link to="/technicians" className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 hover:text-blue-700">
                {t('dashboard.viewAll')} <FaArrowRight className="w-2.5 h-2.5" />
            </Link>
        </div>

        {workers.length === 0 ? (
            <div className="px-6 py-12 text-center">
                <div className="w-14 h-14 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                    <FaUsers className="w-6 h-6 text-gray-300" />
                </div>
                <p className="text-gray-500 text-sm font-medium">
                    {t('dashboard.noWorkersYet', { workers: industryT.workers.toLowerCase() })}
                </p>
                <Link to="/technicians" className="mt-4 inline-flex items-center gap-1.5 text-sm font-semibold text-blue-600 hover:text-blue-700">
                    <FaPlus className="w-3 h-3" />
                    {industryT.addWorker}
                </Link>
            </div>
        ) : (
            <div className="divide-y divide-gray-50">
                {workers.map((worker) => {
                    const status = getWorkerStatus(worker);
                    const statusColor = { available: 'bg-emerald-400', onsite: 'bg-amber-400', offline: 'bg-gray-300' }[status];
                    const statusLabel = {
                        available: t('dashboard.available'),
                        onsite: t('dashboard.onsite'),
                        offline: t('dashboard.offline'),
                    }[status];
                    const initials = (worker.name || '??').split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2);

                    return (
                        <div key={worker.id} className="flex items-center px-6 py-4">
                            <div className="flex-shrink-0 w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center mr-3 shadow-sm">
                                <span className="text-xs font-bold text-white">{initials}</span>
                            </div>
                            <div className="flex-1 min-w-0">
                                <p className="text-sm font-semibold text-gray-900 truncate">{worker.name}</p>
                                <p className="text-xs text-gray-400 truncate">{worker.role || industryT.workerSingle}</p>
                            </div>
                            <div className="flex items-center gap-1.5 flex-shrink-0">
                                <span className={`w-2 h-2 rounded-full ${statusColor}`} />
                                <span className="text-xs text-gray-500">{statusLabel}</span>
                            </div>
                        </div>
                    );
                })}
            </div>
        )}
    </div>
);

// ─── Quick Actions Panel ──────────────────────────────────────────────────────
const QuickActionsPanel = ({ t, industryT, onServiceCall, onNewJob, onAddCustomer, onAddWorker }) => {
    const actions = [
        { onClick: onServiceCall, icon: FaPhone,       label: t('dashboard.newServiceCall'), sub: t('dashboard.newServiceCallSub'), color: 'orange' },
        { onClick: onNewJob,      icon: FaBriefcase,   label: industryT.newJob,               sub: `${t('dashboard.createA')} ${industryT.jobSingle.toLowerCase()}`, color: 'blue' },
        { onClick: onAddCustomer, icon: FaUsers,       label: t('dashboard.addCustomer'),     sub: t('dashboard.addCustomerSub'), color: 'violet' },
        { onClick: onAddWorker,   icon: industryT.icon, label: industryT.addWorker,            sub: industryT.registerWorker, color: 'emerald' },
    ];
    const iconBg = { orange: 'bg-orange-100 text-orange-600', blue: 'bg-blue-100 text-blue-600', violet: 'bg-violet-100 text-violet-600', emerald: 'bg-emerald-100 text-emerald-600' };

    return (
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden h-full">
            <div className="px-6 py-5 border-b border-gray-100">
                <h2 className="text-base font-semibold text-gray-900">{t('dashboard.quickActions')}</h2>
            </div>
            <div className="p-4 space-y-2">
                {actions.map(({ onClick, icon: Icon, label, sub, color }) => (
                    <button
                        key={label}
                        onClick={onClick}
                        className="flex items-center gap-4 w-full px-4 py-3.5 rounded-xl hover:bg-gray-50 transition-colors duration-150 text-left group"
                    >
                        <div className={`w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 ${iconBg[color]}`}>
                            <Icon className="w-4 h-4" />
                        </div>
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">{label}</p>
                            <p className="text-xs text-gray-400 truncate">{sub}</p>
                        </div>
                        <FaArrowRight className="w-3 h-3 text-gray-300 group-hover:text-blue-400 transition-colors flex-shrink-0" />
                    </button>
                ))}
            </div>
        </div>
    );
};

// ─── StatusBadge ──────────────────────────────────────────────────────────────
const StatusBadge = ({ status, t }) => {
    const config = {
        scheduled:   { bg: 'bg-blue-50',    text: 'text-blue-700',    key: 'status.scheduled' },
        in_progress: { bg: 'bg-amber-50',   text: 'text-amber-700',   key: 'status.inProgress' },
        completed:   { bg: 'bg-emerald-50', text: 'text-emerald-700', key: 'status.completed' },
        cancelled:   { bg: 'bg-red-50',     text: 'text-red-700',     key: 'status.cancelled' },
    }[status] || { bg: 'bg-gray-50', text: 'text-gray-700', key: null };

    return (
        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${config.bg} ${config.text}`}>
            {config.key ? t(config.key) : status}
        </span>
    );
};

export default Dashboard;
