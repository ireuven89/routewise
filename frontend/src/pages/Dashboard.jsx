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
} from 'react-icons/fa';
import Layout from '../components/Layout';
import { StatCardSkeleton, CardSkeleton } from '../components/Skeleton';
import { customersAPI, jobsAPI, workersAPI } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';

// ─── Dashboard ───────────────────────────────────────────────────────────────
const Dashboard = () => {
    const { organization } = useAuth();
    const { t } = useLanguage();
    const industry = organization?.industry || 'hvac';
    const isConstruction = industry === 'construction';

    // Industry labels from translations
    const labels = t(`industry.${industry}`, {});
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
    const [loading, setLoading] = useState(true);
    const [copied, setCopied] = useState(false);

    const fetchDashboardData = useCallback(async () => {
        setLoading(true);
        try {
            const [jobsRes, customersRes, workersRes] = await Promise.all([
                jobsAPI.getAll(),
                customersAPI.getAll(),
                workersAPI.getAll(false),
            ]);

            const jobs = jobsRes.data || [];
            const allWorkers = workersRes.data || [];

            const scheduledJobs = jobs.filter((j) => j.status === 'scheduled');
            const activeJobs = jobs.filter((j) => j.status === 'in_progress');
            const completedJobs = jobs.filter((j) => j.status === 'completed');

            // Today's jobs — HVAC only
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
            setWorkers(allWorkers.slice(0, 7));
        } catch (error) {
            console.error('Error fetching dashboard data:', error);
        } finally {
            setLoading(false);
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

    return (
        <Layout>
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">

                {/* ── Company Code Banner ──────────────────────────────────── */}
                {organization?.company_code && (
                    <div className="mb-6 bg-gradient-to-r from-blue-600 to-blue-700 rounded-xl shadow-lg p-5">
                        <div className="flex items-center justify-between flex-wrap gap-3">
                            <div>
                                <p className="text-blue-100 text-sm font-medium">
                                    {t('dashboard.companyCodeDesc', { workers: industryT.workers.toLowerCase() })}
                                </p>
                                <div className="flex items-center mt-2 gap-3">
                                    <span className="text-white text-2xl font-bold font-mono tracking-widest">
                                        {organization.company_code}
                                    </span>
                                    <button
                                        onClick={handleCopyCode}
                                        className="flex items-center gap-1.5 bg-white bg-opacity-20 hover:bg-opacity-30 text-white text-sm font-medium px-3 py-1.5 rounded-lg transition-all duration-150"
                                    >
                                        {copied ? <FaCheck className="w-3.5 h-3.5" /> : <FaCopy className="w-3.5 h-3.5" />}
                                        {copied ? t('dashboard.copied') : t('dashboard.copy')}
                                    </button>
                                </div>
                            </div>
                            <div className="flex items-center gap-2 bg-white bg-opacity-15 rounded-lg px-4 py-2">
                                <industryT.icon className="w-4 h-4 text-blue-200" />
                                <span className="text-white text-sm font-semibold capitalize">{industry}</span>
                            </div>
                        </div>
                    </div>
                )}

                {/* ── Page Header ───────────────────────────────────────────── */}
                <div className="flex justify-between items-center mb-6">
                    <div>
                        <h1 className="text-2xl font-bold text-gray-900">{t('dashboard.title')}</h1>
                        <p className="text-sm text-gray-500 mt-0.5">
                            {t('dashboard.welcome')} {organization?.name || ''}
                        </p>
                    </div>
                    <Link
                        to="/jobs"
                        className="inline-flex items-center px-4 py-2 text-white text-sm font-semibold rounded-lg shadow-sm hover:shadow-md transition-all duration-200 hover:-translate-y-0.5"
                        style={{ backgroundColor: '#ff6b35' }}
                    >
                        <FaPlus className="mr-2 w-3.5 h-3.5" />
                        {industryT.newJob}
                    </Link>
                </div>

                {/* ── Stat Cards ── conditional per industry ────────────────── */}
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4 mb-6">
                    {loading ? (
                        <>
                            <StatCardSkeleton />
                            <StatCardSkeleton />
                            <StatCardSkeleton />
                            <StatCardSkeleton />
                        </>
                    ) : isConstruction ? (
                        <>
                            <StatCard
                                title={t('dashboard.activeProjects')}
                                value={stats.activeJobs}
                                sub={`${stats.scheduledJobs} ${t('dashboard.scheduled')}`}
                                link="/jobs"
                                icon={FaBriefcase}
                                color="blue"
                            />
                            <StatCard
                                title={t('dashboard.completed')}
                                value={stats.completedJobs}
                                sub={t('dashboard.thisQuarter')}
                                link="/jobs?status=completed"
                                icon={FaCheckCircle}
                                color="emerald"
                            />
                            <StatCard
                                title={t('dashboard.totalProjects')}
                                value={stats.totalJobs}
                                sub={t('dashboard.allTime')}
                                link="/jobs"
                                icon={FaCalendarCheck}
                                color="violet"
                            />
                            <StatCard
                                title={industryT.workers}
                                value={stats.totalWorkers}
                                link="/technicians"
                                icon={FaHardHat}
                                color="amber"
                            />
                        </>
                    ) : (
                        <>
                            <StatCard
                                title={t('dashboard.totalJobs')}
                                value={stats.totalJobs}
                                link="/jobs"
                                icon={FaBriefcase}
                                color="blue"
                            />
                            <StatCard
                                title={t('dashboard.scheduledToday')}
                                value={stats.scheduledJobs}
                                link="/jobs?status=scheduled"
                                icon={FaCalendarCheck}
                                color="emerald"
                            />
                            <StatCard
                                title={t('dashboard.customers')}
                                value={stats.totalCustomers}
                                link="/customers"
                                icon={FaUsers}
                                color="violet"
                            />
                            <StatCard
                                title={industryT.workers}
                                value={stats.totalWorkers}
                                link="/technicians"
                                icon={FaWrench}
                                color="amber"
                            />
                        </>
                    )}
                </div>

                {/* ── Main Panels ── conditional per industry ───────────────── */}
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

                    {/* LEFT (2 cols) */}
                    <div className="lg:col-span-2">
                        {loading ? (
                            <CardSkeleton />
                        ) : isConstruction ? (
                            <ActiveWorkersPanel workers={workers} industryT={industryT} t={t} />
                        ) : (
                            <TodaySchedulePanel todayJobs={todayJobs} t={t} />
                        )}
                    </div>

                    {/* RIGHT (1 col): Quick Actions */}
                    <div className="lg:col-span-1">
                        {loading ? (
                            <CardSkeleton />
                        ) : (
                            <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
                                <div className="px-5 py-4 border-b border-gray-100">
                                    <h2 className="text-sm font-semibold text-gray-900">{t('dashboard.quickActions')}</h2>
                                </div>
                                <div className="p-4 space-y-3">
                                    <QuickAction
                                        to="/jobs"
                                        icon={FaBriefcase}
                                        label={industryT.newJob}
                                        sub={`${isConstruction ? t('dashboard.createA') : t('dashboard.createA')} ${industryT.jobSingle.toLowerCase()}`}
                                        color="blue"
                                    />
                                    <QuickAction
                                        to="/customers"
                                        icon={FaUsers}
                                        label={t('dashboard.addCustomer')}
                                        sub={t('dashboard.addCustomerSub')}
                                        color="emerald"
                                    />
                                    <QuickAction
                                        to="/technicians"
                                        icon={industryT.icon}
                                        label={industryT.addWorker}
                                        sub={industryT.registerWorker}
                                        color="violet"
                                    />
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </Layout>
    );
};

// ─── Today's Schedule Panel (HVAC) ───────────────────────────────────────────
const TodaySchedulePanel = ({ todayJobs, t }) => (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-gray-900 flex items-center">
                <FaClock className="mr-2 text-blue-500 w-4 h-4" />
                {t('dashboard.todaySchedule')}
            </h2>
            <Link to="/jobs" className="text-xs text-blue-600 hover:text-blue-700 font-medium">
                {t('dashboard.viewAll')}
            </Link>
        </div>

        {todayJobs.length === 0 ? (
            <div className="px-5 py-10 text-center">
                <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
                    <FaCalendarCheck className="w-5 h-5 text-gray-400" />
                </div>
                <p className="text-gray-500 text-sm">{t('dashboard.noJobsToday')}</p>
                <Link
                    to="/jobs"
                    className="mt-3 inline-flex items-center text-sm text-blue-600 hover:text-blue-700 font-medium"
                >
                    <FaPlus className="mr-1 w-3 h-3" />
                    {t('dashboard.scheduleAJob')}
                </Link>
            </div>
        ) : (
            <div className="divide-y divide-gray-100">
                {todayJobs.map((job) => (
                    <Link
                        key={job.id}
                        to={`/jobs/${job.id}`}
                        className="flex items-center px-5 py-3.5 hover:bg-gray-50 transition-colors duration-100 group"
                    >
                        <div className="flex-shrink-0 w-16 text-center mr-4">
                            <span className="text-xs font-semibold text-blue-600 bg-blue-50 px-2 py-1 rounded-md">
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

// ─── Active Workers Panel (Construction) ─────────────────────────────────────
const getWorkerStatus = (worker) => {
    if (!worker.is_active) return 'offline';
    return 'available';
};

const ActiveWorkersPanel = ({ workers, industryT, t }) => (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-gray-900 flex items-center">
                <FaHardHat className="mr-2 text-amber-500 w-4 h-4" />
                {t('dashboard.activeWorkers', { workers: industryT.workers })}
            </h2>
            <Link to="/technicians" className="text-xs text-blue-600 hover:text-blue-700 font-medium">
                {t('dashboard.viewAll')}
            </Link>
        </div>

        {workers.length === 0 ? (
            <div className="px-5 py-10 text-center">
                <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
                    <FaUsers className="w-5 h-5 text-gray-400" />
                </div>
                <p className="text-gray-500 text-sm">
                    {t('dashboard.noWorkersYet', { workers: industryT.workers.toLowerCase() })}
                </p>
                <Link
                    to="/technicians"
                    className="mt-3 inline-flex items-center text-sm text-blue-600 hover:text-blue-700 font-medium"
                >
                    <FaPlus className="mr-1 w-3 h-3" />
                    {industryT.addWorker}
                </Link>
            </div>
        ) : (
            <div className="divide-y divide-gray-100">
                {workers.map((worker) => {
                    const status = getWorkerStatus(worker);
                    const statusLabel = {
                        available: t('dashboard.available'),
                        onsite: t('dashboard.onsite'),
                        offline: t('dashboard.offline'),
                    }[status];
                    const statusColor = {
                        available: 'bg-emerald-400',
                        onsite: 'bg-amber-400',
                        offline: 'bg-gray-300',
                    }[status];

                    const initials = (worker.name || '??')
                        .split(' ')
                        .map((n) => n[0])
                        .join('')
                        .toUpperCase()
                        .slice(0, 2);

                    return (
                        <div key={worker.id} className="flex items-center px-5 py-3.5">
                            <div className="flex-shrink-0 w-9 h-9 rounded-full bg-blue-100 flex items-center justify-center mr-3">
                                <span className="text-xs font-bold text-blue-700">{initials}</span>
                            </div>
                            <div className="flex-1 min-w-0">
                                <p className="text-sm font-semibold text-gray-900 truncate">{worker.name}</p>
                                <p className="text-xs text-gray-400 truncate">{worker.role || industryT.workerSingle}</p>
                            </div>
                            <div className="flex items-center gap-1.5 flex-shrink-0">
                                <span className={`w-2.5 h-2.5 rounded-full ${statusColor}`} />
                                <span className="text-xs text-gray-500">{statusLabel}</span>
                            </div>
                        </div>
                    );
                })}
            </div>
        )}
    </div>
);

// ─── StatCard ─────────────────────────────────────────────────────────────────
const StatCard = ({ title, value, sub, link, icon: Icon, color }) => {
    const theme = {
        blue:    { bg: 'bg-blue-50',    icon: 'text-blue-600',    border: 'border-blue-200' },
        emerald: { bg: 'bg-emerald-50', icon: 'text-emerald-600', border: 'border-emerald-200' },
        violet:  { bg: 'bg-violet-50',  icon: 'text-violet-600',  border: 'border-violet-200' },
        amber:   { bg: 'bg-amber-50',   icon: 'text-amber-600',   border: 'border-amber-200' },
    }[color] || { bg: 'bg-gray-50', icon: 'text-gray-600', border: 'border-gray-200' };

    return (
        <Link
            to={link}
            className={`bg-white rounded-xl shadow-sm border ${theme.border} p-4 hover:shadow-md transition-all duration-200 hover:-translate-y-0.5`}
        >
            <div className="flex items-center justify-between">
                <div>
                    <p className="text-xs font-medium text-gray-500 uppercase tracking-wide">{title}</p>
                    <p className="text-2xl font-bold text-gray-900 mt-1">{value}</p>
                    {sub && <p className="text-xs text-gray-400 mt-0.5">{sub}</p>}
                </div>
                <div className={`${theme.bg} rounded-lg p-2.5`}>
                    <Icon className={`w-5 h-5 ${theme.icon}`} />
                </div>
            </div>
        </Link>
    );
};

// ─── QuickAction ──────────────────────────────────────────────────────────────
const QuickAction = ({ to, icon: Icon, label, sub, color }) => {
    const theme = {
        blue:    { icon: 'text-blue-600',    bg: 'bg-blue-50',    hover: 'hover:bg-blue-100' },
        emerald: { icon: 'text-emerald-600', bg: 'bg-emerald-50', hover: 'hover:bg-emerald-100' },
        violet:  { icon: 'text-violet-600',  bg: 'bg-violet-50',  hover: 'hover:bg-violet-100' },
    }[color];

    return (
        <Link
            to={to}
            className={`flex items-center gap-3 p-3 rounded-lg ${theme.bg} ${theme.hover} transition-colors duration-150`}
        >
            <div className="flex-shrink-0">
                <Icon className={`w-4 h-4 ${theme.icon}`} />
            </div>
            <div className="min-w-0">
                <p className="text-sm font-semibold text-gray-900">{label}</p>
                <p className="text-xs text-gray-500 truncate">{sub}</p>
            </div>
        </Link>
    );
};

// ─── StatusBadge ──────────────────────────────────────────────────────────────
const StatusBadge = ({ status, t }) => {
    const config = {
        scheduled:   { bg: 'bg-blue-100',    text: 'text-blue-700',    key: 'status.scheduled' },
        in_progress: { bg: 'bg-amber-100',   text: 'text-amber-700',   key: 'status.inProgress' },
        completed:   { bg: 'bg-emerald-100', text: 'text-emerald-700', key: 'status.completed' },
        cancelled:   { bg: 'bg-red-100',     text: 'text-red-700',     key: 'status.cancelled' },
    }[status] || { bg: 'bg-gray-100', text: 'text-gray-700', key: null };

    return (
        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${config.bg} ${config.text}`}>
            {config.key ? t(config.key) : status}
        </span>
    );
};

export default Dashboard;
