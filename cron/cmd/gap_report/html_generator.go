package main

import (
	"fmt"
	"strings"
)

func generateHTMLContent(reports []GapReport, summary ReportSummary) string {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nugs Collection Analytics • Professional Dashboard</title>
    
    <!-- Premium Fonts -->
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800;900&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
    
    <!-- Chart.js & GSAP for animations -->
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/gsap/3.12.2/gsap.min.js"></script>
    
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        :root {
            --primary: #6366f1;
            --primary-light: #818cf8;
            --primary-dark: #4f46e5;
            --secondary: #a855f7;
            --success: #22c55e;
            --warning: #eab308;
            --danger: #ef4444;
            --dark: #0f0f23;
            --dark-secondary: #1a1a2e;
            --dark-tertiary: #232345;
            --light: #ffffff;
            --gray: #9ca3af;
            --glass: rgba(255, 255, 255, 0.02);
            --glass-border: rgba(255, 255, 255, 0.08);
        }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--dark);
            color: var(--light);
            min-height: 100vh;
            overflow-x: hidden;
            position: relative;
        }

        /* Stunning animated gradient background */
        body::before {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            width: 200%;
            height: 200%;
            background: 
                radial-gradient(circle at 20% 50%, rgba(99, 102, 241, 0.15) 0%, transparent 40%),
                radial-gradient(circle at 80% 80%, rgba(168, 85, 247, 0.15) 0%, transparent 40%),
                radial-gradient(circle at 40% 20%, rgba(34, 197, 94, 0.1) 0%, transparent 40%),
                radial-gradient(circle at 90% 10%, rgba(239, 68, 68, 0.1) 0%, transparent 40%);
            animation: gradientShift 30s ease-in-out infinite;
            z-index: -2;
        }

        @keyframes gradientShift {
            0%, 100% { transform: rotate(0deg) translate(-25%, -25%); }
            33% { transform: rotate(120deg) translate(-25%, -25%) scale(1.1); }
            66% { transform: rotate(240deg) translate(-25%, -25%) scale(0.9); }
        }

        /* Particle effect */
        .particles {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            pointer-events: none;
            z-index: -1;
        }

        .particle {
            position: absolute;
            width: 2px;
            height: 2px;
            background: var(--primary-light);
            border-radius: 50%;
            opacity: 0;
            animation: float 20s infinite;
        }

        @keyframes float {
            0%, 100% { 
                opacity: 0;
                transform: translateY(100vh) translateX(0);
            }
            10%, 90% { opacity: 0.4; }
            50% { opacity: 0.8; }
        }

        /* Header */
        .header {
            backdrop-filter: blur(20px);
            background: linear-gradient(135deg, var(--glass) 0%, rgba(99, 102, 241, 0.03) 100%);
            border-bottom: 1px solid var(--glass-border);
            padding: 2rem 0;
            position: sticky;
            top: 0;
            z-index: 100;
            animation: slideDown 0.6s ease-out;
        }

        @keyframes slideDown {
            from { transform: translateY(-100%); opacity: 0; }
            to { transform: translateY(0); opacity: 1; }
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 0 2rem;
        }

        .header-content {
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 2rem;
        }

        .logo-section {
            display: flex;
            align-items: center;
            gap: 1.5rem;
        }

        .logo {
            width: 56px;
            height: 56px;
            background: linear-gradient(135deg, var(--primary) 0%, var(--secondary) 100%);
            border-radius: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 28px;
            box-shadow: 0 10px 30px rgba(99, 102, 241, 0.3);
            animation: pulse 3s ease-in-out infinite;
        }

        @keyframes pulse {
            0%, 100% { transform: scale(1); box-shadow: 0 10px 30px rgba(99, 102, 241, 0.3); }
            50% { transform: scale(1.05); box-shadow: 0 15px 40px rgba(99, 102, 241, 0.5); }
        }

        .logo-text h1 {
            font-size: 1.875rem;
            font-weight: 800;
            background: linear-gradient(135deg, var(--light) 0%, var(--gray) 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            letter-spacing: -0.03em;
        }

        .logo-text p {
            font-size: 0.925rem;
            color: var(--gray);
            margin-top: 0.25rem;
            font-weight: 500;
        }

        .header-actions {
            display: flex;
            gap: 1rem;
        }

        .btn {
            padding: 0.75rem 1.5rem;
            border-radius: 12px;
            font-weight: 600;
            font-size: 0.925rem;
            border: none;
            cursor: pointer;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            position: relative;
            overflow: hidden;
        }

        .btn::before {
            content: '';
            position: absolute;
            top: 50%;
            left: 50%;
            width: 0;
            height: 0;
            border-radius: 50%;
            background: rgba(255, 255, 255, 0.2);
            transform: translate(-50%, -50%);
            transition: width 0.6s, height 0.6s;
        }

        .btn:hover::before {
            width: 300px;
            height: 300px;
        }

        .btn-primary {
            background: linear-gradient(135deg, var(--primary) 0%, var(--primary-dark) 100%);
            color: white;
            box-shadow: 0 4px 15px rgba(99, 102, 241, 0.3);
        }

        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 25px rgba(99, 102, 241, 0.4);
        }

        .btn-secondary {
            background: var(--glass);
            color: var(--primary-light);
            border: 1px solid var(--glass-border);
        }

        .btn-secondary:hover {
            background: rgba(99, 102, 241, 0.1);
            border-color: var(--primary);
        }

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
            gap: 1.5rem;
            margin: 3rem 0;
            animation: fadeInUp 0.8s ease-out 0.2s both;
        }

        @keyframes fadeInUp {
            from { opacity: 0; transform: translateY(30px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .stat-card {
            background: linear-gradient(135deg, var(--glass) 0%, rgba(99, 102, 241, 0.02) 100%);
            backdrop-filter: blur(10px);
            border: 1px solid var(--glass-border);
            border-radius: 20px;
            padding: 2rem;
            position: relative;
            overflow: hidden;
            transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .stat-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 3px;
            background: linear-gradient(90deg, var(--primary), var(--secondary));
            transform: scaleX(0);
            transform-origin: left;
            transition: transform 0.4s ease;
        }

        .stat-card:hover {
            transform: translateY(-5px) scale(1.02);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
            border-color: var(--primary);
        }

        .stat-card:hover::before {
            transform: scaleX(1);
        }

        .stat-icon {
            width: 56px;
            height: 56px;
            border-radius: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 28px;
            margin-bottom: 1.5rem;
            position: relative;
        }

        .stat-icon::after {
            content: '';
            position: absolute;
            inset: -2px;
            border-radius: 16px;
            padding: 2px;
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
            -webkit-mask-composite: xor;
            mask-composite: exclude;
            opacity: 0;
            transition: opacity 0.4s ease;
        }

        .stat-card:hover .stat-icon::after {
            opacity: 1;
        }

        .stat-value {
            font-size: 2.5rem;
            font-weight: 800;
            background: linear-gradient(135deg, var(--light) 0%, var(--gray) 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            line-height: 1;
            margin-bottom: 0.5rem;
        }

        .stat-label {
            font-size: 0.875rem;
            color: var(--gray);
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .stat-trend {
            display: inline-flex;
            align-items: center;
            gap: 0.25rem;
            margin-top: 1rem;
            padding: 0.375rem 0.875rem;
            border-radius: 999px;
            font-size: 0.875rem;
            font-weight: 600;
        }

        .stat-trend.positive {
            background: rgba(34, 197, 94, 0.1);
            color: var(--success);
        }

        .stat-trend.negative {
            background: rgba(239, 68, 68, 0.1);
            color: var(--danger);
        }

        /* Charts Section */
        .charts-section {
            margin: 4rem 0;
            animation: fadeInUp 0.8s ease-out 0.4s both;
        }

        .section-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
        }

        .section-title {
            font-size: 1.75rem;
            font-weight: 700;
            background: linear-gradient(135deg, var(--light) 0%, var(--gray) 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .charts-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
            gap: 2rem;
        }

        .chart-card {
            background: linear-gradient(135deg, var(--glass) 0%, rgba(99, 102, 241, 0.02) 100%);
            backdrop-filter: blur(10px);
            border: 1px solid var(--glass-border);
            border-radius: 24px;
            padding: 2rem;
            transition: all 0.4s ease;
        }

        .chart-card:hover {
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.2);
            transform: translateY(-2px);
        }

        .chart-header {
            font-size: 1.25rem;
            font-weight: 600;
            margin-bottom: 1.5rem;
            color: var(--light);
        }

        .chart-container {
            position: relative;
            height: 350px;
        }

        /* Search & Filter */
        .controls {
            background: linear-gradient(135deg, var(--glass) 0%, rgba(99, 102, 241, 0.02) 100%);
            backdrop-filter: blur(10px);
            border: 1px solid var(--glass-border);
            border-radius: 24px;
            padding: 2rem;
            margin: 3rem 0;
            animation: fadeInUp 0.8s ease-out 0.6s both;
        }

        .controls-grid {
            display: grid;
            grid-template-columns: 2fr 1fr 1fr;
            gap: 1.5rem;
        }

        .search-container {
            position: relative;
        }

        .search-input {
            width: 100%;
            padding: 1rem 1.25rem 1rem 3.5rem;
            background: var(--dark);
            border: 2px solid var(--glass-border);
            border-radius: 14px;
            color: var(--light);
            font-size: 1rem;
            font-weight: 500;
            transition: all 0.3s ease;
        }

        .search-input:focus {
            outline: none;
            border-color: var(--primary);
            box-shadow: 0 0 0 4px rgba(99, 102, 241, 0.1);
        }

        .search-icon {
            position: absolute;
            left: 1.25rem;
            top: 50%;
            transform: translateY(-50%);
            color: var(--gray);
        }

        .select-input {
            padding: 1rem 1.25rem;
            background: var(--dark);
            border: 2px solid var(--glass-border);
            border-radius: 14px;
            color: var(--light);
            font-size: 1rem;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.3s ease;
        }

        .select-input:focus {
            outline: none;
            border-color: var(--primary);
            box-shadow: 0 0 0 4px rgba(99, 102, 241, 0.1);
        }

        /* Artists Table */
        .table-container {
            background: linear-gradient(135deg, var(--glass) 0%, rgba(99, 102, 241, 0.02) 100%);
            backdrop-filter: blur(10px);
            border: 1px solid var(--glass-border);
            border-radius: 24px;
            overflow: hidden;
            margin: 3rem 0;
            animation: fadeInUp 0.8s ease-out 0.8s both;
        }

        .table-header {
            padding: 2rem;
            background: var(--dark);
            border-bottom: 1px solid var(--glass-border);
        }

        .table-title {
            font-size: 1.5rem;
            font-weight: 600;
            color: var(--light);
        }

        .artists-table {
            width: 100%;
            border-collapse: collapse;
        }

        .artists-table th {
            padding: 1.25rem 1.5rem;
            text-align: left;
            font-weight: 600;
            font-size: 0.875rem;
            color: var(--gray);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            background: var(--dark);
            border-bottom: 1px solid var(--glass-border);
            cursor: pointer;
            transition: all 0.2s ease;
            position: sticky;
            top: 0;
            z-index: 10;
        }

        .artists-table th:hover {
            color: var(--primary-light);
        }

        .artists-table td {
            padding: 1.5rem;
            border-bottom: 1px solid var(--glass-border);
            font-size: 0.975rem;
        }

        .artists-table tbody tr {
            transition: all 0.3s ease;
            cursor: pointer;
            position: relative;
        }

        .artists-table tbody tr::before {
            content: '';
            position: absolute;
            left: 0;
            top: 0;
            width: 3px;
            height: 100%;
            background: var(--primary);
            transform: scaleY(0);
            transition: transform 0.3s ease;
        }

        .artists-table tbody tr:hover {
            background: rgba(99, 102, 241, 0.05);
        }

        .artists-table tbody tr:hover::before {
            transform: scaleY(1);
        }

        .artist-cell {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .artist-avatar {
            width: 48px;
            height: 48px;
            border-radius: 12px;
            background: linear-gradient(135deg, var(--primary) 0%, var(--secondary) 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 700;
            font-size: 1.125rem;
            color: white;
            box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
        }

        .artist-name {
            font-weight: 600;
            color: var(--light);
        }

        .progress-cell {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .progress-bar {
            flex: 1;
            height: 10px;
            background: var(--dark);
            border-radius: 999px;
            overflow: hidden;
            position: relative;
        }

        .progress-fill {
            height: 100%;
            border-radius: 999px;
            transition: width 1s cubic-bezier(0.4, 0, 0.2, 1);
            position: relative;
            overflow: hidden;
        }

        .progress-fill::after {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.4), transparent);
            animation: shimmer 2s infinite;
        }

        @keyframes shimmer {
            0% { transform: translateX(-100%); }
            100% { transform: translateX(100%); }
        }

        .progress-text {
            min-width: 65px;
            text-align: right;
            font-weight: 700;
            font-size: 0.925rem;
        }

        .badge {
            display: inline-flex;
            align-items: center;
            padding: 0.5rem 1rem;
            border-radius: 999px;
            font-size: 0.875rem;
            font-weight: 600;
        }

        .badge-success {
            background: rgba(34, 197, 94, 0.1);
            color: var(--success);
            border: 1px solid rgba(34, 197, 94, 0.2);
        }

        .badge-warning {
            background: rgba(234, 179, 8, 0.1);
            color: var(--warning);
            border: 1px solid rgba(234, 179, 8, 0.2);
        }

        .badge-danger {
            background: rgba(239, 68, 68, 0.1);
            color: var(--danger);
            border: 1px solid rgba(239, 68, 68, 0.2);
        }

        /* Modal */
        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.9);
            backdrop-filter: blur(10px);
            z-index: 1000;
            animation: fadeIn 0.3s ease;
        }

        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }

        .modal-content {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            width: 90%;
            max-width: 900px;
            max-height: 90vh;
            background: linear-gradient(135deg, var(--dark-secondary) 0%, var(--dark-tertiary) 100%);
            backdrop-filter: blur(20px);
            border: 1px solid var(--glass-border);
            border-radius: 28px;
            overflow: hidden;
            animation: slideUp 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        }

        @keyframes slideUp {
            from {
                opacity: 0;
                transform: translate(-50%, -40%);
            }
            to {
                opacity: 1;
                transform: translate(-50%, -50%);
            }
        }

        .modal-header {
            padding: 2.5rem;
            background: var(--dark);
            border-bottom: 1px solid var(--glass-border);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .modal-title {
            font-size: 1.75rem;
            font-weight: 700;
            color: var(--light);
        }

        .modal-close {
            width: 48px;
            height: 48px;
            border-radius: 12px;
            background: var(--glass);
            border: 1px solid var(--glass-border);
            color: var(--gray);
            display: flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            transition: all 0.3s ease;
            font-size: 1.5rem;
        }

        .modal-close:hover {
            background: rgba(239, 68, 68, 0.2);
            color: var(--danger);
            transform: rotate(90deg);
        }

        .modal-body {
            padding: 2.5rem;
            overflow-y: auto;
            max-height: calc(90vh - 200px);
        }

        .shows-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
            gap: 1.25rem;
        }

        .show-card {
            background: var(--glass);
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            padding: 1.25rem;
            transition: all 0.3s ease;
            cursor: pointer;
        }

        .show-card:hover {
            background: rgba(99, 102, 241, 0.1);
            border-color: var(--primary);
            transform: translateY(-3px);
            box-shadow: 0 10px 20px rgba(0, 0, 0, 0.2);
        }

        .show-date {
            font-weight: 700;
            color: var(--primary-light);
            margin-bottom: 0.5rem;
            font-size: 1.05rem;
        }

        .show-venue {
            color: var(--light);
            font-size: 0.975rem;
            margin-bottom: 0.25rem;
            font-weight: 500;
        }

        .show-location {
            color: var(--gray);
            font-size: 0.875rem;
        }

        .show-id {
            display: inline-block;
            margin-top: 0.75rem;
            padding: 0.375rem 0.75rem;
            background: rgba(99, 102, 241, 0.1);
            border: 1px solid rgba(99, 102, 241, 0.2);
            border-radius: 8px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.8rem;
            color: var(--primary-light);
            font-weight: 600;
        }

        /* Footer */
        .footer {
            margin-top: 6rem;
            padding: 3rem 0;
            border-top: 1px solid var(--glass-border);
            text-align: center;
            color: var(--gray);
            font-size: 0.925rem;
        }

        .footer-content {
            display: flex;
            justify-content: center;
            align-items: center;
            gap: 0.5rem;
        }

        /* Responsive */
        @media (max-width: 768px) {
            .controls-grid {
                grid-template-columns: 1fr;
            }
            
            .charts-grid {
                grid-template-columns: 1fr;
            }
            
            .stats-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <!-- Particle Effect -->
    <div class="particles" id="particles"></div>

    <header class="header">
        <div class="container">
            <div class="header-content">
                <div class="logo-section">
                    <div class="logo">🎵</div>
                    <div class="logo-text">
                        <h1>Nugs Collection Analytics</h1>
                        <p>Professional Concert Archive Dashboard</p>
                    </div>
                </div>
                <div class="header-actions">
                    <button class="btn btn-secondary" onclick="exportJSON()">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                            <polyline points="7 10 12 15 17 10"></polyline>
                            <line x1="12" y1="15" x2="12" y2="3"></line>
                        </svg>
                        Export JSON
                    </button>
                    <button class="btn btn-primary" onclick="exportCSV()">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                            <polyline points="7 10 12 15 17 10"></polyline>
                            <line x1="12" y1="15" x2="12" y2="3"></line>
                        </svg>
                        Export CSV
                    </button>
                </div>
            </div>
        </div>
    </header>

    <main class="container">
        <!-- Stats Grid -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-icon" style="background: linear-gradient(135deg, rgba(99, 102, 241, 0.2), rgba(168, 85, 247, 0.2));">
                    📊
                </div>
                <div class="stat-value">` + fmt.Sprintf("%d", summary.TotalArtists) + `</div>
                <div class="stat-label">Total Artists</div>
                <div class="stat-trend positive">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
                    </svg>
                    Actively Monitored
                </div>
            </div>
            
            <div class="stat-card">
                <div class="stat-icon" style="background: linear-gradient(135deg, rgba(34, 197, 94, 0.2), rgba(16, 185, 129, 0.2));">
                    ✅
                </div>
                <div class="stat-value">` + fmt.Sprintf("%d", summary.TotalShowsHave) + `</div>
                <div class="stat-label">Shows Downloaded</div>
                <div class="stat-trend positive">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
                    </svg>
                    In Collection
                </div>
            </div>
            
            <div class="stat-card">
                <div class="stat-icon" style="background: linear-gradient(135deg, rgba(234, 179, 8, 0.2), rgba(251, 191, 36, 0.2));">
                    📀
                </div>
                <div class="stat-value">` + fmt.Sprintf("%d", summary.TotalShowsAvail) + `</div>
                <div class="stat-label">Shows Available</div>
                <div class="stat-trend positive">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
                    </svg>
                    On Platform
                </div>
            </div>
            
            <div class="stat-card">
                <div class="stat-icon" style="background: linear-gradient(135deg, rgba(99, 102, 241, 0.2), rgba(79, 70, 229, 0.2));">
                    📈
                </div>
                <div class="stat-value">` + fmt.Sprintf("%.1f%%", summary.OverallCompletion) + `</div>
                <div class="stat-label">Completion Rate</div>
                <div class="stat-trend positive">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
                    </svg>
                    Overall Progress
                </div>
            </div>
            
            <div class="stat-card">
                <div class="stat-icon" style="background: linear-gradient(135deg, rgba(239, 68, 68, 0.2), rgba(220, 38, 38, 0.2));">
                    ❌
                </div>
                <div class="stat-value">` + fmt.Sprintf("%d", summary.TotalMissing) + `</div>
                <div class="stat-label">Missing Shows</div>
                <div class="stat-trend negative">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="23 18 13.5 8.5 8.5 13.5 1 6"></polyline>
                    </svg>
                    To Download
                </div>
            </div>
        </div>

        <!-- Charts Section -->
        <div class="charts-section">
            <div class="section-header">
                <h2 class="section-title">📊 Collection Analytics</h2>
            </div>
            
            <div class="charts-grid">
                <div class="chart-card">
                    <h3 class="chart-header">Completion Rate by Artist</h3>
                    <div class="chart-container">
                        <canvas id="completionChart"></canvas>
                    </div>
                </div>
                
                <div class="chart-card">
                    <h3 class="chart-header">Missing Shows Distribution</h3>
                    <div class="chart-container">
                        <canvas id="missingChart"></canvas>
                    </div>
                </div>
            </div>
        </div>

        <!-- Search and Filter Controls -->
        <div class="controls">
            <div class="controls-grid">
                <div class="search-container">
                    <svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="11" cy="11" r="8"></circle>
                        <path d="m21 21-4.35-4.35"></path>
                    </svg>
                    <input type="text" 
                           class="search-input" 
                           placeholder="Search artists..." 
                           id="searchInput"
                           oninput="filterArtists()">
                </div>
                
                <select class="select-input" id="sortSelect" onchange="sortArtists()">
                    <option value="artist">Sort by Artist</option>
                    <option value="completion">Sort by Completion</option>
                    <option value="missing">Sort by Missing</option>
                    <option value="total">Sort by Total Shows</option>
                </select>
                
                <select class="select-input" id="filterSelect" onchange="filterArtists()">
                    <option value="all">All Artists</option>
                    <option value="complete">100% Complete</option>
                    <option value="incomplete">Has Missing</option>
                    <option value="critical">< 50% Complete</option>
                </select>
            </div>
        </div>

        <!-- Artists Table -->
        <div class="table-container">
            <div class="table-header">
                <h2 class="table-title">🎤 Artist Collection Details</h2>
            </div>
            
            <table class="artists-table">
                <thead>
                    <tr>
                        <th onclick="sortBy('artist')">Artist</th>
                        <th onclick="sortBy('total')">Total</th>
                        <th onclick="sortBy('downloaded')">Downloaded</th>
                        <th onclick="sortBy('completion')">Completion</th>
                        <th onclick="sortBy('missing')">Missing</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody id="artistsTableBody">
`

	// Add artist rows
	for _, report := range reports {
		completionClass := "badge-danger"
		completionColor := "#ef4444"
		if report.CompletionPct >= 100 {
			completionClass = "badge-success"
			completionColor = "#22c55e"
		} else if report.CompletionPct >= 75 {
			completionClass = "badge-warning"
			completionColor = "#eab308"
		} else if report.CompletionPct >= 50 {
			completionColor = "#a855f7"
		}

		firstLetter := strings.ToUpper(string(report.Artist[0]))

		html += fmt.Sprintf(`
                    <tr class="artist-row" data-artist="%s">
                        <td>
                            <div class="artist-cell">
                                <div class="artist-avatar">%s</div>
                                <span class="artist-name">%s</span>
                            </div>
                        </td>
                        <td>%d</td>
                        <td>%d</td>
                        <td>
                            <div class="progress-cell">
                                <div class="progress-bar">
                                    <div class="progress-fill" style="width: %.1f%%; background: linear-gradient(90deg, %s, %s);"></div>
                                </div>
                                <span class="progress-text" style="color: %s;">%.1f%%</span>
                            </div>
                        </td>
                        <td>
                            <span class="badge %s">%d shows</span>
                        </td>
                        <td>
                            <button class="btn btn-secondary" onclick="showMissingDetails('%s', %d)" style="padding: 0.6rem 1.2rem; font-size: 0.875rem;">
                                View Details
                            </button>
                        </td>
                    </tr>`,
			report.Artist, firstLetter, report.Artist,
			report.TotalAvailable, report.TotalDownloaded,
			report.CompletionPct, completionColor, completionColor,
			completionColor, report.CompletionPct,
			completionClass, report.MissingCount,
			strings.ReplaceAll(report.Artist, "'", "\\'"), report.ArtistID)
	}

	html += `
                </tbody>
            </table>
        </div>
    </main>

    <!-- Modal -->
    <div id="missingModal" class="modal">
        <div class="modal-content">
            <div class="modal-header">
                <h2 class="modal-title" id="modalTitle">Missing Shows</h2>
                <div class="modal-close" onclick="closeModal()">×</div>
            </div>
            <div class="modal-body" id="modalBody">
                <!-- Content will be inserted here -->
            </div>
        </div>
    </div>

    <footer class="footer">
        <div class="container">
            <div class="footer-content">
                <span>🎵</span>
                <span>Nugs Collection Gap Report</span>
                <span>•</span>
                <span>Generated with precision analytics</span>
            </div>
        </div>
    </footer>

    <script>
        // Create particles
        function createParticles() {
            const particlesContainer = document.getElementById('particles');
            for (let i = 0; i < 50; i++) {
                const particle = document.createElement('div');
                particle.className = 'particle';
                particle.style.left = Math.random() * 100 + '%';
                particle.style.animationDelay = Math.random() * 20 + 's';
                particle.style.animationDuration = (15 + Math.random() * 10) + 's';
                particlesContainer.appendChild(particle);
            }
        }
        createParticles();

        // Data
        const artistsData = [`

	// Add artist data
	for i, report := range reports {
		if i > 0 {
			html += `,`
		}
		html += fmt.Sprintf(`
            {
                "artist": "%s",
                "artist_id": %d,
                "total_available": %d,
                "total_downloaded": %d,
                "completion_pct": %.1f,
                "missing_count": %d,
                "missing_shows": [`,
			strings.ReplaceAll(report.Artist, `"`, `\"`),
			report.ArtistID,
			report.TotalAvailable,
			report.TotalDownloaded,
			report.CompletionPct,
			report.MissingCount)

		for j, show := range report.MissingShows {
			if j > 0 {
				html += `,`
			}
			html += fmt.Sprintf(`
                    {
                        "container_id": %d,
                        "date": "%s",
                        "venue": "%s",
                        "city": "%s",
                        "state": "%s"
                    }`,
				show.ContainerID,
				strings.ReplaceAll(show.Date, `"`, `\"`),
				strings.ReplaceAll(show.Venue, `"`, `\"`),
				strings.ReplaceAll(show.City, `"`, `\"`),
				strings.ReplaceAll(show.State, `"`, `\"`))
		}

		html += `
                ]
            }`
	}

	html += `
        ];

        let filteredData = [...artistsData];
        let currentSort = 'artist';
        let sortDirection = 'asc';

        // Initialize charts
        function initCharts() {
            // Completion Chart
            const topArtists = [...artistsData]
                .sort((a, b) => b.completion_pct - a.completion_pct)
                .slice(0, 10);

            const completionCtx = document.getElementById('completionChart').getContext('2d');
            new Chart(completionCtx, {
                type: 'bar',
                data: {
                    labels: topArtists.map(a => a.artist.length > 20 ? a.artist.substring(0, 20) + '...' : a.artist),
                    datasets: [{
                        data: topArtists.map(a => a.completion_pct),
                        backgroundColor: topArtists.map(a => {
                            if (a.completion_pct >= 90) return 'rgba(34, 197, 94, 0.8)';
                            if (a.completion_pct >= 70) return 'rgba(234, 179, 8, 0.8)';
                            if (a.completion_pct >= 50) return 'rgba(168, 85, 247, 0.8)';
                            return 'rgba(239, 68, 68, 0.8)';
                        }),
                        borderColor: topArtists.map(a => {
                            if (a.completion_pct >= 90) return 'rgba(34, 197, 94, 1)';
                            if (a.completion_pct >= 70) return 'rgba(234, 179, 8, 1)';
                            if (a.completion_pct >= 50) return 'rgba(168, 85, 247, 1)';
                            return 'rgba(239, 68, 68, 1)';
                        }),
                        borderWidth: 2,
                        borderRadius: 10
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            display: false
                        },
                        tooltip: {
                            backgroundColor: 'rgba(15, 15, 35, 0.95)',
                            borderColor: 'rgba(99, 102, 241, 0.5)',
                            borderWidth: 1,
                            titleColor: '#fff',
                            bodyColor: '#9ca3af',
                            padding: 14,
                            borderRadius: 10,
                            displayColors: false,
                            callbacks: {
                                label: function(context) {
                                    return context.parsed.y.toFixed(1) + '% Complete';
                                }
                            }
                        }
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            max: 100,
                            grid: {
                                color: 'rgba(255, 255, 255, 0.05)',
                                drawBorder: false
                            },
                            ticks: {
                                color: '#9ca3af',
                                callback: function(value) {
                                    return value + '%';
                                }
                            }
                        },
                        x: {
                            grid: {
                                display: false
                            },
                            ticks: {
                                color: '#9ca3af',
                                maxRotation: 45,
                                minRotation: 45
                            }
                        }
                    }
                }
            });

            // Missing Shows Chart
            const missingCtx = document.getElementById('missingChart').getContext('2d');
            new Chart(missingCtx, {
                type: 'doughnut',
                data: {
                    labels: ['Downloaded', 'Missing'],
                    datasets: [{
                        data: [` + fmt.Sprintf("%d, %d", summary.TotalShowsHave, summary.TotalMissing) + `],
                        backgroundColor: [
                            'rgba(34, 197, 94, 0.8)',
                            'rgba(239, 68, 68, 0.8)'
                        ],
                        borderColor: [
                            'rgba(34, 197, 94, 1)',
                            'rgba(239, 68, 68, 1)'
                        ],
                        borderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    cutout: '70%',
                    plugins: {
                        legend: {
                            position: 'bottom',
                            labels: {
                                color: '#9ca3af',
                                padding: 20,
                                font: {
                                    size: 14
                                }
                            }
                        },
                        tooltip: {
                            backgroundColor: 'rgba(15, 15, 35, 0.95)',
                            borderColor: 'rgba(99, 102, 241, 0.5)',
                            borderWidth: 1,
                            titleColor: '#fff',
                            bodyColor: '#9ca3af',
                            padding: 14,
                            borderRadius: 10,
                            callbacks: {
                                label: function(context) {
                                    const label = context.label || '';
                                    const value = context.parsed || 0;
                                    const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                    const percentage = ((value / total) * 100).toFixed(1);
                                    return label + ': ' + value.toLocaleString() + ' (' + percentage + '%)';
                                }
                            }
                        }
                    }
                }
            });
        }

        // Initialize on load
        document.addEventListener('DOMContentLoaded', function() {
            initCharts();
            animateStats();
        });

        // Animate stats
        function animateStats() {
            gsap.from(".stat-card", {
                duration: 0.8,
                y: 30,
                opacity: 0,
                stagger: 0.1,
                ease: "power2.out"
            });

            gsap.from(".stat-value", {
                duration: 1.5,
                textContent: 0,
                snap: { textContent: 1 },
                stagger: 0.1,
                ease: "power2.out"
            });
        }

        // Filter artists
        function filterArtists() {
            const searchTerm = document.getElementById('searchInput').value.toLowerCase();
            const filterType = document.getElementById('filterSelect').value;
            
            filteredData = artistsData.filter(artist => {
                const matchesSearch = artist.artist.toLowerCase().includes(searchTerm);
                
                let matchesFilter = true;
                switch(filterType) {
                    case 'complete':
                        matchesFilter = artist.completion_pct === 100;
                        break;
                    case 'incomplete':
                        matchesFilter = artist.completion_pct < 100;
                        break;
                    case 'critical':
                        matchesFilter = artist.completion_pct < 50;
                        break;
                }
                
                return matchesSearch && matchesFilter;
            });
            
            renderTable();
        }

        // Sort artists
        function sortBy(field) {
            if (currentSort === field) {
                sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
            } else {
                currentSort = field;
                sortDirection = 'asc';
            }
            
            sortArtists();
        }

        function sortArtists() {
            const sortType = document.getElementById('sortSelect').value || currentSort;
            
            filteredData.sort((a, b) => {
                let compareValue = 0;
                
                switch(sortType) {
                    case 'artist':
                        compareValue = a.artist.localeCompare(b.artist);
                        break;
                    case 'completion':
                        compareValue = a.completion_pct - b.completion_pct;
                        break;
                    case 'missing':
                        compareValue = a.missing_count - b.missing_count;
                        break;
                    case 'total':
                        compareValue = a.total_available - b.total_available;
                        break;
                    case 'downloaded':
                        compareValue = a.total_downloaded - b.total_downloaded;
                        break;
                }
                
                return sortDirection === 'asc' ? compareValue : -compareValue;
            });
            
            renderTable();
        }

        // Render table
        function renderTable() {
            const tbody = document.getElementById('artistsTableBody');
            tbody.innerHTML = '';
            
            filteredData.forEach(artist => {
                const completionClass = artist.completion_pct >= 100 ? 'badge-success' :
                                      artist.completion_pct >= 75 ? 'badge-warning' : 'badge-danger';
                const completionColor = artist.completion_pct >= 100 ? '#22c55e' :
                                       artist.completion_pct >= 75 ? '#eab308' :
                                       artist.completion_pct >= 50 ? '#a855f7' : '#ef4444';
                
                const firstLetter = artist.artist[0].toUpperCase();
                
                const row = document.createElement('tr');
                row.className = 'artist-row';
                row.dataset.artist = artist.artist;
                row.innerHTML = ` + "`" + `
                    <td>
                        <div class="artist-cell">
                            <div class="artist-avatar">${firstLetter}</div>
                            <span class="artist-name">${artist.artist}</span>
                        </div>
                    </td>
                    <td>${artist.total_available}</td>
                    <td>${artist.total_downloaded}</td>
                    <td>
                        <div class="progress-cell">
                            <div class="progress-bar">
                                <div class="progress-fill" style="width: ${artist.completion_pct}%; background: linear-gradient(90deg, ${completionColor}, ${completionColor});"></div>
                            </div>
                            <span class="progress-text" style="color: ${completionColor};">${artist.completion_pct.toFixed(1)}%</span>
                        </div>
                    </td>
                    <td>
                        <span class="badge ${completionClass}">${artist.missing_count} shows</span>
                    </td>
                    <td>
                        <button class="btn btn-secondary" onclick="showMissingDetails('${artist.artist.replace(/'/g, "\\'")}', ${artist.artist_id})" style="padding: 0.6rem 1.2rem; font-size: 0.875rem;">
                            View Details
                        </button>
                    </td>
                ` + "`" + `;
                tbody.appendChild(row);
            });
        }

        // Show missing details
        function showMissingDetails(artistName, artistId) {
            const artist = artistsData.find(a => a.artist === artistName);
            if (!artist) return;
            
            document.getElementById('modalTitle').textContent = artistName + ' - Missing Shows (' + artist.missing_count + ')';
            
            let content = '<div class="shows-grid">';
            artist.missing_shows.forEach(show => {
                content += ` + "`" + `
                    <div class="show-card">
                        <div class="show-date">${show.date}</div>
                        <div class="show-venue">${show.venue}</div>
                        <div class="show-location">${show.city}, ${show.state}</div>
                        <span class="show-id">#${show.container_id}</span>
                    </div>
                ` + "`" + `;
            });
            content += '</div>';
            
            document.getElementById('modalBody').innerHTML = content;
            document.getElementById('missingModal').style.display = 'block';
            
            // Animate modal content
            gsap.from(".show-card", {
                duration: 0.4,
                y: 20,
                opacity: 0,
                stagger: 0.05,
                ease: "power2.out"
            });
        }

        // Close modal
        function closeModal() {
            document.getElementById('missingModal').style.display = 'none';
        }

        // Export functions
        function exportJSON() {
            const data = {
                summary: {
                    total_artists: ` + fmt.Sprintf("%d", summary.TotalArtists) + `,
                    total_shows_have: ` + fmt.Sprintf("%d", summary.TotalShowsHave) + `,
                    total_shows_available: ` + fmt.Sprintf("%d", summary.TotalShowsAvail) + `,
                    overall_completion: ` + fmt.Sprintf("%.1f", summary.OverallCompletion) + `,
                    total_missing: ` + fmt.Sprintf("%d", summary.TotalMissing) + `
                },
                artists: artistsData
            };
            
            const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'nugs-collection-report.json';
            a.click();
        }

        function exportCSV() {
            let csv = 'Artist,Total Available,Total Downloaded,Completion %,Missing Count\\n';
            artistsData.forEach(artist => {
                csv += ` + "`" + `"${artist.artist}",${artist.total_available},${artist.total_downloaded},${artist.completion_pct.toFixed(1)},${artist.missing_count}\\n` + "`" + `;
            });
            
            const blob = new Blob([csv], { type: 'text/csv' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'nugs-collection-report.csv';
            a.click();
        }

        // Close modal on outside click
        window.onclick = function(event) {
            const modal = document.getElementById('missingModal');
            if (event.target == modal) {
                modal.style.display = 'none';
            }
        }
    </script>
</body>
</html>`

	return html
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
